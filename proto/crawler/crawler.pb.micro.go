// Code generated by protoc-gen-micro. DO NOT EDIT.
// source: crawler.proto

package crawler

import (
	fmt "fmt"
	empty "github.com/golang/protobuf/ptypes/empty"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	proto "google.golang.org/protobuf/proto"
	math "math"
)

import (
	context "context"
	client "go-micro.dev/v4/client"
	server "go-micro.dev/v4/server"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ client.Option
var _ server.Option

// Client API for CrawlerMaster service

type CrawlerMasterService interface {
	AddResource(ctx context.Context, in *ResourceSpec, opts ...client.CallOption) (*NodeSpec, error)
	DeleteResource(ctx context.Context, in *ResourceSpec, opts ...client.CallOption) (*empty.Empty, error)
}

type crawlerMasterService struct {
	c    client.Client
	name string
}

func NewCrawlerMasterService(name string, c client.Client) CrawlerMasterService {
	return &crawlerMasterService{
		c:    c,
		name: name,
	}
}

func (c *crawlerMasterService) AddResource(ctx context.Context, in *ResourceSpec, opts ...client.CallOption) (*NodeSpec, error) {
	req := c.c.NewRequest(c.name, "CrawlerMaster.AddResource", in)
	out := new(NodeSpec)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *crawlerMasterService) DeleteResource(ctx context.Context, in *ResourceSpec, opts ...client.CallOption) (*empty.Empty, error) {
	req := c.c.NewRequest(c.name, "CrawlerMaster.DeleteResource", in)
	out := new(empty.Empty)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for CrawlerMaster service

type CrawlerMasterHandler interface {
	AddResource(context.Context, *ResourceSpec, *NodeSpec) error
	DeleteResource(context.Context, *ResourceSpec, *empty.Empty) error
}

func RegisterCrawlerMasterHandler(s server.Server, hdlr CrawlerMasterHandler, opts ...server.HandlerOption) error {
	type crawlerMaster interface {
		AddResource(ctx context.Context, in *ResourceSpec, out *NodeSpec) error
		DeleteResource(ctx context.Context, in *ResourceSpec, out *empty.Empty) error
	}
	type CrawlerMaster struct {
		crawlerMaster
	}
	h := &crawlerMasterHandler{hdlr}
	return s.Handle(s.NewHandler(&CrawlerMaster{h}, opts...))
}

type crawlerMasterHandler struct {
	CrawlerMasterHandler
}

func (h *crawlerMasterHandler) AddResource(ctx context.Context, in *ResourceSpec, out *NodeSpec) error {
	return h.CrawlerMasterHandler.AddResource(ctx, in, out)
}

func (h *crawlerMasterHandler) DeleteResource(ctx context.Context, in *ResourceSpec, out *empty.Empty) error {
	return h.CrawlerMasterHandler.DeleteResource(ctx, in, out)
}
