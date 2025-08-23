package executionrepo

import (
	"context"
	"errors"
	"time"

	domain "github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
	"github.com/yitter/idgenerator-go/idgen"
	"gorm.io/gorm"
)

type MysqlRepositoryImpl struct {
	commonrepo.DefaultRepo
}

func NewMysqlRepositoryImpl(db commonrepo.DB) domain.Repo {
	return &MysqlRepositoryImpl{
		DefaultRepo: commonrepo.NewDefaultRepo(db),
	}
}

// Count implements execution.Repo.
func (r *MysqlRepositoryImpl) Count(ctx context.Context, query domain.CountQuery) (int64, error) {
	var db = r.Db(ctx).Model(&TaskExecution{})

	if query.StartTime.IsPresent() {
		db = db.Where("scheduled_time >= ?", query.StartTime.MustGet())
	}
	if query.EndTime.IsPresent() {
		db = db.Where("scheduled_time <= ?", query.EndTime.MustGet())
	}
	if query.TaskID.IsPresent() {
		db = db.Where("task_id = ?", query.TaskID.MustGet())
	}
	if query.Status.IsPresent() {
		db = db.Where("status = ?", query.Status.MustGet())
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *MysqlRepositoryImpl) GetByID(ctx context.Context, id uint64) (*domain.TaskExecution, error) {
	var po = new(TaskExecution)
	if err := r.Db(ctx).First(po, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

func (r *MysqlRepositoryImpl) Save(ctx context.Context, execution *domain.TaskExecution) error {
	po := new(TaskExecution).FromDomain(execution)
	err := r.Db(ctx).Save(po).Error
	if err != nil {
		return err
	}
	execution.ID = po.ID
	execution.CreatedAt = po.CreatedAt
	execution.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *MysqlRepositoryImpl) Create(ctx context.Context, execution *domain.TaskExecution) error {
	po := new(TaskExecution).FromDomain(execution)
	err := r.Db(ctx).Create(po).Error
	if err != nil {
		return err
	}
	execution.ID = po.ID
	execution.CreatedAt = po.CreatedAt
	execution.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *MysqlRepositoryImpl) Delete(ctx context.Context, id uint64) error {
	return r.Db(ctx).Delete(&TaskExecution{}, id).Error
}

func (r *MysqlRepositoryImpl) List(ctx context.Context, filter domain.ListFilter, offset, limit int) ([]*domain.TaskExecution, int64, error) {
	db := r.Db(ctx).Model(&TaskExecution{})

	if filter.StartTime.IsPresent() {
		db = db.Where("scheduled_time >= ?", filter.StartTime.MustGet())
	}
	if filter.EndTime.IsPresent() {
		db = db.Where("scheduled_time <= ?", filter.EndTime.MustGet())
	}
	if filter.TaskID.IsPresent() {
		db = db.Where("task_id = ?", filter.TaskID.MustGet())
	}
	if filter.Status.IsPresent() {
		db = db.Where("status = ?", filter.Status.MustGet())
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	var pos []*TaskExecution
	if err := db.Order("scheduled_time DESC").Limit(limit).Offset(offset).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	domains := make([]*domain.TaskExecution, len(pos))
	for i := range pos {
		domains[i] = pos[i].ToDomain()
	}
	return domains, count, nil
}

func (r *MysqlRepositoryImpl) CountByTaskAndStatus(ctx context.Context, taskID uint64, statuses []domain.ExecutionStatus) (int64, error) {
	var count int64
	err := r.Db(ctx).Model(&TaskExecution{}).
		Where("task_id = ?", taskID).
		Where("status IN ?", statuses).
		Count(&count).Error
	return count, err
}

func (r *MysqlRepositoryImpl) CountByExecutorAndStatus(ctx context.Context, executorID uint64, statuses []domain.ExecutionStatus) (int64, error) {
	var count int64
	err := r.Db(ctx).Model(&TaskExecution{}).
		Where("executor_id = ?", executorID).
		Where("status IN ?", statuses).
		Count(&count).Error
	return count, err
}

func (r *MysqlRepositoryImpl) CreateSkipped(ctx context.Context, taskID uint64, reason string) (*domain.TaskExecution, error) {
	execution_ := &TaskExecution{
		Mode: commonrepo.Mode{
			ID: uint64(idgen.NextId()),
		},
		TaskID:        taskID,
		ExecutorID:    0,
		ScheduledTime: time.Now(),
		StartTime:     nil,
		EndTime:       nil,
		Status:        domain.ExecutionStatusSkipped,
		Result:        nil,
		Logs:          reason,
		RetryCount:    0,
	}
	if err := r.Db(ctx).Create(execution_).Error; err != nil {
		return nil, err
	}
	return execution_.ToDomain(), nil
}

func (r *MysqlRepositoryImpl) CountByTaskAndTimeRange(ctx context.Context, taskID uint64, startTime, endTime time.Time) (int64, error) {
	var count int64
	err := r.Db(ctx).Model(&TaskExecution{}).
		Where("task_id = ?", taskID).
		Where("created_at >= ?", startTime).
		Where("created_at < ?", endTime).
		Count(&count).Error
	return count, err
}

func (r *MysqlRepositoryImpl) CountByTaskStatusAndTimeRange(ctx context.Context, taskID uint64, status domain.ExecutionStatus, startTime, endTime time.Time) (int64, error) {
	var count int64
	err := r.Db(ctx).Model(&TaskExecution{}).
		Where("task_id = ?", taskID).
		Where("status = ?", status).
		Where("created_at >= ?", startTime).
		Where("created_at < ?", endTime).
		Count(&count).Error
	return count, err
}

func (r *MysqlRepositoryImpl) CountByTaskStatusesAndTimeRange(ctx context.Context, taskID uint64, statuses []domain.ExecutionStatus, startTime, endTime time.Time) (int64, error) {
	var count int64
	err := r.Db(ctx).Model(&TaskExecution{}).
		Where("task_id = ?", taskID).
		Where("status IN ?", statuses).
		Where("created_at >= ?", startTime).
		Where("created_at < ?", endTime).
		Count(&count).Error
	return count, err
}

func (r *MysqlRepositoryImpl) GetAvgDuration(ctx context.Context, taskID uint64, startTime time.Time) (float64, error) {
	var avgDuration *float64
	err := r.Db(ctx).Model(&TaskExecution{}).
		Where("task_id = ?", taskID).
		Where("created_at >= ?", startTime).
		Where("start_time IS NOT NULL").
		Where("end_time IS NOT NULL").
		Select("AVG(TIMESTAMPDIFF(SECOND, start_time, end_time))").
		Scan(&avgDuration).Error
	
	if err != nil {
		return 0, err
	}
	
	// 如果没有匹配的记录，AVG返回NULL，返回0
	if avgDuration == nil {
		return 0, nil
	}
	
	return *avgDuration, nil
}
