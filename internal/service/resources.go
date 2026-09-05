package service

import (
	"context"
	"errors"
	"time"

	"github.com/gcc798/ai-ops-gateway/internal/pagination"
	"github.com/gcc798/ai-ops-gateway/internal/resources"
)

func (s *Service) ListResources(ctx context.Context, kind string, f resources.Filter) (pagination.Result[resources.Resource], error) {
	if err := f.Validate(); err != nil {
		return pagination.Result[resources.Resource]{}, err
	}
	if s.Resources == nil {
		return pagination.Result[resources.Resource]{Items: []resources.Resource{}, Page: f.Page, PageSize: f.PageSize}, nil
	}
	return s.Resources.List(ctx, kind, f)
}
func (s *Service) Resource(ctx context.Context, kind, name string) (resources.Resource, error) {
	if s.Resources == nil {
		return resources.Resource{}, errors.New("resource store unavailable")
	}
	return s.Resources.Get(ctx, kind, name)
}

// 使用缓存客户端前检查最新配置，资源删除后不能继续使用旧连接。
func (s *Service) syncResource(kind, name string) error {
	if s.Resources == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.resourceMu.Lock()
	defer s.resourceMu.Unlock()
	r, err := s.Resources.Get(ctx, kind, name)
	key := kind + ":" + name
	rev := r.Revision()
	if err != nil || (s.revisions[key] != "" && s.revisions[key] != rev) {
		switch kind {
		case "database":
			s.Databases.Remove(name)
		case "linux":
			s.Linux.Remove(name)
		case "kubernetes":
			s.Kubernetes.Remove(name)
		}
	}
	if err != nil {
		delete(s.revisions, key)
		return err
	}
	if s.revisions == nil {
		s.revisions = map[string]string{}
	}
	s.revisions[key] = rev
	return nil
}
func (s *Service) operationResource(ctx context.Context, kind, name string) (resources.Resource, error) {
	if s.Resources == nil {
		return resources.Resource{Name: name}, nil
	}
	return s.Resource(ctx, kind, name)
}
