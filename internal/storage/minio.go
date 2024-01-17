package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/cecobask/spacelift-coding-challenge/internal/container"
	"github.com/cecobask/spacelift-coding-challenge/pkg/log"
	"github.com/lafikl/consistent"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
)

const (
	minioApiPort         = 9000
	minioRootPasswordKey = "MINIO_ROOT_PASSWORD"
	minioRootUserKey     = "MINIO_ROOT_USER"
	networkName          = "spacelift"
	nodeLabel            = "minio.storage.node=true"
)

var (
	ErrObjectNotFound = errors.New("object not found")
)

type Node struct {
	id     string
	client *minio.Client
}

type Minio struct {
	ctx    context.Context
	docker *container.Docker
	logger *log.Logger
	nodes  map[string]Node
	ring   *consistent.Consistent
}

func NewMinio(ctx context.Context, logger *log.Logger) (*Minio, error) {
	docker, err := container.NewDocker(ctx, logger)
	if err != nil {
		return nil, fmt.Errorf("could not create docker client: %w", err)
	}
	m := &Minio{
		ctx:    ctx,
		docker: docker,
		logger: logger,
		nodes:  make(map[string]Node),
		ring:   consistent.New(),
	}
	if err = m.setup(); err != nil {
		return nil, fmt.Errorf("could not setup minio: %w", err)
	}
	return m, nil
}

func (m *Minio) setup() error {
	containers, err := m.docker.GetContainersWithLabel(nodeLabel)
	if err != nil {
		return err
	}
	network, err := m.docker.GetNetworkWithName(networkName)
	if err != nil {
		return err
	}
	for _, c := range containers {
		endpoint := fmt.Sprintf("%s:%d", c.NetworkSettings.Networks[network.Name].IPAddress, minioApiPort)
		variables := container.GetContainerEnvironmentVariables(c.Config)
		client, err := minio.New(endpoint, &minio.Options{
			Creds: credentials.NewStaticV4(
				variables[minioRootUserKey],
				variables[minioRootPasswordKey],
				"",
			),
		})
		if err != nil {
			return err
		}
		n := Node{
			id:     c.ID[:12],
			client: client,
		}
		if err = n.setupDefaultBucket(m.ctx); err != nil {
			return err
		}
		m.nodes[n.id] = n
		m.ring.Add(n.id)
	}
	return nil
}

func (n *Node) setupDefaultBucket(ctx context.Context) error {
	exists, err := n.client.BucketExists(ctx, n.id)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return n.client.MakeBucket(ctx, n.id, minio.MakeBucketOptions{})
}

func (m *Minio) GetObject(id string) ([]byte, *minio.ObjectInfo, error) {
	nodeID, err := m.ring.Get(id)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find any nodes in the ring: %w", err)
	}
	node, ok := m.nodes[nodeID]
	if !ok {
		return nil, nil, fmt.Errorf("could not find node with id %s", nodeID)
	}
	object, stat, err := node.GetObject(m.ctx, id)
	if err != nil {
		return nil, nil, err
	}
	defer object.Close()
	data, err := io.ReadAll(object)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read object: %w", err)
	}
	m.logger.Info("object was retrieved", "id", id, "stat", stat)
	return data, stat, nil
}

func (m *Minio) PutObject(id string, data []byte, contentType string) error {
	nodeID, err := m.ring.Get(id)
	if err != nil {
		return fmt.Errorf("could not find any nodes in the ring: %w", err)
	}
	node, ok := m.nodes[nodeID]
	if !ok {
		return fmt.Errorf("could not find node with id %s", nodeID)
	}
	info, err := node.PutObject(m.ctx, id, data, contentType)
	if err != nil {
		return err
	}
	m.logger.Info("object was uploaded", "id", id, "info", info)
	return nil
}

func (n *Node) GetObject(ctx context.Context, name string) (*minio.Object, *minio.ObjectInfo, error) {
	object, err := n.client.GetObject(ctx, n.id, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, err
	}
	stat, err := object.Stat()
	if err != nil {
		errorResponse := minio.ToErrorResponse(err)
		if errorResponse.Code == "NoSuchKey" {
			return nil, nil, ErrObjectNotFound
		}
		return nil, nil, err
	}
	return object, &stat, nil
}

func (n *Node) PutObject(ctx context.Context, id string, data []byte, contentType string) (*minio.UploadInfo, error) {
	info, err := n.client.PutObject(ctx, n.id, id, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, err
	}
	return &info, nil
}
