package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/data_models"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/logger"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/pkg/tasker"
	"gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/storage"
)

const (
	interruptedTaskFinishError = "任务因程序退出而中断，请重新发起"
	expiredApprovalFinishError = "等待确认的工具请求已过期，请重新发起"
)

func finalizeRunningAssistantArtifacts(message *data_models.Message, finishedAt time.Time) {
	if message == nil {
		return
	}
	if message.AssistantMessageExtra == nil {
		message.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}

	for idx := range message.AssistantMessageExtra.ToolUses {
		toolUse := &message.AssistantMessageExtra.ToolUses[idx]
		if toolUse.Status != data_models.ToolUseStatusRunning &&
			toolUse.Status != data_models.ToolUseStatusPending &&
			toolUse.Status != data_models.ToolUseStatusAwaitingApproval {
			continue
		}
		if toolUse.StartedAt == nil {
			toolUse.StartedAt = &finishedAt
		}
		toolUse.FinishedAt = &finishedAt
		toolUse.ElapsedMs = finishedAt.Sub(*toolUse.StartedAt).Milliseconds()
		toolUse.Status = data_models.ToolUseStatusError
	}

	for idx := range message.AssistantMessageExtra.ExecutionTrace.Steps {
		step := &message.AssistantMessageExtra.ExecutionTrace.Steps[idx]
		if step.Status != data_models.TraceStepStatusRunning &&
			step.Status != data_models.TraceStepStatusPending &&
			step.Status != data_models.TraceStepStatusAwaitingApproval {
			continue
		}
		if step.StartedAt == nil {
			step.StartedAt = &finishedAt
		}
		step.FinishedAt = &finishedAt
		step.ElapsedMs = finishedAt.Sub(*step.StartedAt).Milliseconds()
		step.Status = data_models.TraceStepStatusError
	}
}

func markAssistantMessageInterrupted(message *data_models.Message, finishError string, finishedAt time.Time) {
	if message == nil {
		return
	}
	if message.AssistantMessageExtra == nil {
		message.AssistantMessageExtra = &data_models.AssistantMessageExtra{}
	}

	finalizeRunningAssistantArtifacts(message, finishedAt)
	message.AssistantMessageExtra.PendingApprovals = nil
	message.AssistantMessageExtra.CurrentStage = ""
	message.AssistantMessageExtra.CurrentAgent = ""
	message.AssistantMessageExtra.FinishReason = "error"
	message.AssistantMessageExtra.FinishError = finishError
}

func (s *Service) hasLiveTaskRuntime(taskUUID string) bool {
	if taskUUID == "" {
		return false
	}
	_, ok := tasker.Manager.GetTaskRuntime(taskUUID)
	return ok
}

func (s *Service) reconcileInterruptedTask(ctx context.Context, task data_models.Task, finishError string) error {
	if task.TaskUuid == "" {
		return nil
	}
	if task.Status != data_models.TaskStatusPending &&
		task.Status != data_models.TaskStatusRunning &&
		task.Status != data_models.TaskStatusWaitingApproval {
		return nil
	}

	if finishError == "" {
		finishError = interruptedTaskFinishError
	}
	finishedAt := time.Now()
	task.Status = data_models.TaskStatusFailed
	task.FinishReason = "error"
	task.FinishError = finishError
	task.FinishedAt = &finishedAt

	return s.storage.NewFnTransaction(ctx, func(ctx context.Context, tx *storage.Storage) error {
		if err := tx.SaveTask(ctx, task); err != nil {
			return err
		}

		approvals, err := tx.ListToolApprovalsByTaskUUID(ctx, task.TaskUuid)
		if err != nil {
			return err
		}
		for _, approval := range approvals {
			if approval.Status != data_models.ToolApprovalStatusPending {
				continue
			}
			approval.Status = data_models.ToolApprovalStatusExpired
			if approval.Decision == "" {
				approval.Decision = data_models.ToolApprovalDecisionReject
			}
			approval.ResponseComment = finishError
			approval.RespondedAt = &finishedAt
			if err := tx.SaveToolApproval(ctx, approval); err != nil {
				return err
			}
		}

		if task.AssistantMessageUuid == "" {
			return nil
		}
		message, err := tx.GetMessageByUUID(ctx, task.AssistantMessageUuid)
		if err != nil {
			return err
		}
		if message == nil {
			return nil
		}

		markAssistantMessageInterrupted(message, finishError, finishedAt)
		return tx.SaveOrUpdateMessage(ctx, *message)
	})
}

func (s *Service) repairStaleActiveTask(ctx context.Context, task *data_models.Task) (*data_models.Task, error) {
	if task == nil {
		return nil, nil
	}
	if task.Status != data_models.TaskStatusPending &&
		task.Status != data_models.TaskStatusRunning &&
		task.Status != data_models.TaskStatusWaitingApproval {
		return task, nil
	}
	if s.hasLiveTaskRuntime(task.TaskUuid) {
		return task, nil
	}
	finishError := interruptedTaskFinishError
	if task.Status == data_models.TaskStatusWaitingApproval {
		finishError = expiredApprovalFinishError
	}
	if err := s.reconcileInterruptedTask(ctx, *task, finishError); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Service) recoverStaleRunningTasks(ctx context.Context) error {
	tasks, err := s.storage.GetRunningTasks(ctx)
	if err != nil {
		return err
	}

	var reconcileErrs []error
	for _, task := range tasks {
		if s.hasLiveTaskRuntime(task.TaskUuid) {
			continue
		}
		finishError := interruptedTaskFinishError
		if task.Status == data_models.TaskStatusWaitingApproval {
			finishError = expiredApprovalFinishError
		}
		if err := s.reconcileInterruptedTask(ctx, task, finishError); err != nil {
			reconcileErrs = append(reconcileErrs, fmt.Errorf("task %s: %w", task.TaskUuid, err))
		}
	}
	return errors.Join(reconcileErrs...)
}

func (s *Service) ServiceShutdown() error {
	var shutdownErrs []error
	for _, runtime := range tasker.Manager.ListRunningTasks() {
		task, err := s.storage.GetTask(context.Background(), runtime.TaskUUID)
		if err != nil {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("get task %s: %w", runtime.TaskUUID, err))
			continue
		}
		if task == nil {
			continue
		}
		if err := s.reconcileInterruptedTask(context.Background(), *task, interruptedTaskFinishError); err != nil {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("shutdown reconcile %s: %w", runtime.TaskUUID, err))
			continue
		}
	}
	err := errors.Join(shutdownErrs...)
	if err != nil {
		logger.Error("service shutdown reconcile task error", err)
	}
	return err
}
