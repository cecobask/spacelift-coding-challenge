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

var ErrObjectNotFound = errors.New("object not found")

type node struct {
	id     string
	client *minio.Client
}

type Minio struct {
	docker *container.Docker
	nodes  map[string]node
	ring   *consistent.Consistent
}

func NewMinio(docker *container.Docker) *Minio {
	return &Minio{
		docker: docker,
		nodes:  make(map[string]node),
		ring:   consistent.New(),
	}
}

func (m *Minio) Setup(ctx context.Context) error {
	log.FromContext(ctx).Debug("setting up minio")
	containers, err := m.docker.GetContainersWithLabel(ctx, nodeLabel)
	if err != nil {
		return err
	}
	network, err := m.docker.GetNetworkWithName(ctx, networkName)
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
		n := node{
			id:     c.ID[:12],
			client: client,
		}
		if err = n.setupDefaultBucket(ctx); err != nil {
			return err
		}
		m.nodes[n.id] = n
		m.ring.Add(n.id)
	}
	return nil
}

func (m *Minio) GetObject(ctx context.Context, id string) ([]byte, *minio.ObjectInfo, error) {
	log.FromContext(ctx).Debug("getting object", "id", id)
	n, err := m.findNodeForObject(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	object, stat, err := n.getObject(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	defer object.Close()
	data, err := io.ReadAll(object)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read object: %w", err)
	}
	log.FromContext(ctx).Debug("object was retrieved", "id", stat.Key)
	return data, stat, nil
}

func (m *Minio) PutObject(ctx context.Context, id string, data []byte, contentType string) error {
	log.FromContext(ctx).Debug("putting object", "id", id, "contentType", contentType)
	n, err := m.findNodeForObject(ctx, id)
	if err != nil {
		return err
	}
	info, err := n.putObject(ctx, id, data, contentType)
	if err != nil {
		return err
	}
	log.FromContext(ctx).Debug("object was uploaded", "id", info.Key, "bucket", info.Bucket)
	return nil
}

func (m *Minio) findNodeForObject(ctx context.Context, id string) (*node, error) {
	log.FromContext(ctx).Debug("finding node for object", "object.id", id)
	nodeID, err := m.ring.Get(id)
	if err != nil {
		return nil, fmt.Errorf("could not find any nodes in the ring: %w", err)
	}
	n, ok := m.nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("could not find node with id %s", nodeID)
	}
	return &n, nil
}

func (n *node) setupDefaultBucket(ctx context.Context) error {
	log.FromContext(ctx).Debug("setting up default bucket", "node.id", n.id)
	exists, err := n.client.BucketExists(ctx, n.id)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return n.client.MakeBucket(ctx, n.id, minio.MakeBucketOptions{})
}

func (n *node) getObject(ctx context.Context, name string) (*minio.Object, *minio.ObjectInfo, error) {
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

func (n *node) putObject(ctx context.Context, id string, data []byte, contentType string) (*minio.UploadInfo, error) {
	info, err := n.client.PutObject(ctx, n.id, id, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, err
	}
	return &info, nil
}
