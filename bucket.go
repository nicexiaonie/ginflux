package ginflux

import (
	"context"
	"fmt"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// BucketManager Bucket 管理器
type BucketManager struct {
	client *Client
}

// NewBucketManager 创建 Bucket 管理器
func NewBucketManager(client *Client) *BucketManager {
	return &BucketManager{client: client}
}

// CreateBucket 创建 Bucket
func (bm *BucketManager) CreateBucket(ctx context.Context, name string, retentionHours int) (*domain.Bucket, error) {
	orgAPI := bm.client.OrganizationsAPI()
	org, err := orgAPI.FindOrganizationByName(ctx, bm.client.config.Organization)
	if err != nil {
		return nil, fmt.Errorf("failed to find organization: %w", err)
	}

	bucketAPI := bm.client.BucketsAPI()
	retention := int64(retentionHours * 3600)

	bucket, err := bucketAPI.CreateBucketWithName(ctx, org, name, domain.RetentionRule{
		EverySeconds: retention,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return bucket, nil
}

// GetBucket 获取 Bucket
func (bm *BucketManager) GetBucket(ctx context.Context, name string) (*domain.Bucket, error) {
	bucketAPI := bm.client.BucketsAPI()
	bucket, err := bucketAPI.FindBucketByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find bucket: %w", err)
	}
	return bucket, nil
}

// ListBuckets 列出所有 Buckets
func (bm *BucketManager) ListBuckets(ctx context.Context) (*[]domain.Bucket, error) {
	bucketAPI := bm.client.BucketsAPI()
	buckets, err := bucketAPI.GetBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	return buckets, nil
}

// DeleteBucket 删除 Bucket
func (bm *BucketManager) DeleteBucket(ctx context.Context, name string) error {
	bucket, err := bm.GetBucket(ctx, name)
	if err != nil {
		return err
	}

	bucketAPI := bm.client.BucketsAPI()
	err = bucketAPI.DeleteBucket(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

// UpdateBucketRetention 更新 Bucket 保留策略
func (bm *BucketManager) UpdateBucketRetention(ctx context.Context, name string, retentionHours int) (*domain.Bucket, error) {
	bucket, err := bm.GetBucket(ctx, name)
	if err != nil {
		return nil, err
	}

	retention := int64(retentionHours * 3600)
	bucket.RetentionRules = []domain.RetentionRule{
		{EverySeconds: retention},
	}

	bucketAPI := bm.client.BucketsAPI()
	updatedBucket, err := bucketAPI.UpdateBucket(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to update bucket: %w", err)
	}

	return updatedBucket, nil
}

// DeleteData 删除指定时间范围的数据
func (bm *BucketManager) DeleteData(ctx context.Context, bucket, measurement string, start, stop time.Time, predicate string) error {
	orgAPI := bm.client.OrganizationsAPI()
	org, err := orgAPI.FindOrganizationByName(ctx, bm.client.config.Organization)
	if err != nil {
		return fmt.Errorf("failed to find organization: %w", err)
	}

	bucketObj, err := bm.GetBucket(ctx, bucket)
	if err != nil {
		return err
	}

	deleteAPI := bm.client.DeleteAPI()

	// 构建删除谓词
	deletePredicate := fmt.Sprintf(`_measurement="%s"`, measurement)
	if predicate != "" {
		deletePredicate = fmt.Sprintf(`%s AND %s`, deletePredicate, predicate)
	}

	err = deleteAPI.DeleteWithName(ctx, *org.Id, *bucketObj.Id, start, stop, deletePredicate)
	if err != nil {
		return fmt.Errorf("failed to delete data: %w", err)
	}

	return nil
}
