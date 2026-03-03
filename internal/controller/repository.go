package controller

import (
	"context"
	"sync"
	"time"

	pb "github.com/psds-microservice/api-gateway/pkg/gen"
	"google.golang.org/protobuf/proto"
)

// StreamStore — абстракция хранилища стримов (Dependency Inversion).
type StreamStore interface {
	SaveStream(ctx context.Context, streamID string, stream *pb.ActiveStream)
	GetStream(ctx context.Context, streamID string) *pb.ActiveStream
	GetStats(ctx context.Context, streamID string) *pb.StreamStats
	UpdateStats(ctx context.Context, streamID string, frame *pb.VideoFrame) *pb.StreamStats
	RemoveStream(ctx context.Context, streamID string)
	GetAllActiveStreams(ctx context.Context) []*pb.ActiveStream
	GetAllStreams(ctx context.Context) []*pb.ActiveStream
	GetAllStats(ctx context.Context) []*pb.StreamStats
}

// ClientStore — абстракция хранилища клиентов.
type ClientStore interface {
	SaveClient(ctx context.Context, client *pb.ClientInfo)
	GetClient(ctx context.Context, clientID string) *pb.ClientInfo
	RemoveClient(ctx context.Context, clientID string)
	GetAllClients(ctx context.Context) []*pb.ClientInfo
}

// ClientRepository — in-memory репозиторий клиентов
type ClientRepository struct {
	clients map[string]*pb.ClientInfo
	mu      sync.RWMutex
}

// NewClientRepository создает новый репозиторий
func NewClientRepository() *ClientRepository {
	return &ClientRepository{
		clients: make(map[string]*pb.ClientInfo),
	}
}

func (r *ClientRepository) SaveClient(ctx context.Context, client *pb.ClientInfo) {
	if client == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if client.Stats == nil {
		client.Stats = &pb.ClientInfo_ClientStats{}
	}
	client.Stats.LastActivity = time.Now().Unix()
	r.clients[client.ClientId] = client
}

// GetClient returns a copy of the client so callers cannot race with SaveClient/RemoveClient.
func (r *ClientRepository) GetClient(ctx context.Context, clientID string) *pb.ClientInfo {
	r.mu.RLock()
	c := r.clients[clientID]
	r.mu.RUnlock()
	if c == nil {
		return nil
	}
	return proto.Clone(c).(*pb.ClientInfo)
}

func (r *ClientRepository) RemoveClient(ctx context.Context, clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, clientID)
}

// GetAllClients returns copies so callers cannot race with SaveClient/RemoveClient.
func (r *ClientRepository) GetAllClients(ctx context.Context) []*pb.ClientInfo {
	r.mu.RLock()
	out := make([]*pb.ClientInfo, 0, len(r.clients))
	for _, client := range r.clients {
		out = append(out, proto.Clone(client).(*pb.ClientInfo))
	}
	r.mu.RUnlock()
	return out
}

// StreamRepository — in-memory репозиторий стримов
type StreamRepository struct {
	streams map[string]*pb.ActiveStream
	stats   map[string]*pb.StreamStats
	mu      sync.RWMutex
}

// NewStreamRepository создает новый репозиторий
func NewStreamRepository() *StreamRepository {
	return &StreamRepository{
		streams: make(map[string]*pb.ActiveStream),
		stats:   make(map[string]*pb.StreamStats),
	}
}

func (r *StreamRepository) SaveStream(ctx context.Context, streamID string, stream *pb.ActiveStream) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streams[streamID] = stream
	if _, exists := r.stats[streamID]; !exists {
		r.stats[streamID] = &pb.StreamStats{
			StreamId:       streamID,
			ClientId:       stream.ClientId,
			StartTime:      time.Now().Unix(),
			FramesReceived: 0,
			BytesReceived:  0,
			AverageFps:     0.0,
			CurrentFps:     0.0,
			Width:          1920,
			Height:         1080,
			Codec:          "H.264",
			IsRecording:    stream.IsRecording,
			IsStreaming:    stream.IsStreaming,
		}
	}
}

func (r *StreamRepository) UpdateStats(ctx context.Context, streamID string, frame *pb.VideoFrame) *pb.StreamStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	stats, exists := r.stats[streamID]
	if !exists {
		return nil
	}
	stats.FramesReceived++
	if frame != nil {
		stats.BytesReceived += int64(len(frame.FrameData))
		if frame.Width > 0 {
			stats.Width = frame.Width
		}
		if frame.Height > 0 {
			stats.Height = frame.Height
		}
		now := time.Now().Unix()
		duration := float64(now - stats.StartTime)
		if duration > 0 {
			stats.AverageFps = float32(float64(stats.FramesReceived) / duration)
		}
		stats.Duration = now - stats.StartTime
	}
	return stats
}

func (r *StreamRepository) GetStats(ctx context.Context, streamID string) *pb.StreamStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats[streamID]
}

func (r *StreamRepository) GetAllStats(ctx context.Context) []*pb.StreamStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	allStats := make([]*pb.StreamStats, 0, len(r.stats))
	for _, stats := range r.stats {
		allStats = append(allStats, stats)
	}
	return allStats
}

func (r *StreamRepository) GetStream(ctx context.Context, streamID string) *pb.ActiveStream {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.streams[streamID]
}

func (r *StreamRepository) GetAllStreams(ctx context.Context) []*pb.ActiveStream {
	r.mu.RLock()
	defer r.mu.RUnlock()
	streams := make([]*pb.ActiveStream, 0, len(r.streams))
	for _, stream := range r.streams {
		streams = append(streams, stream)
	}
	return streams
}

func (r *StreamRepository) RemoveStream(ctx context.Context, streamID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.streams, streamID)
	delete(r.stats, streamID)
}

func (r *StreamRepository) GetAllActiveStreams(ctx context.Context) []*pb.ActiveStream {
	r.mu.RLock()
	defer r.mu.RUnlock()
	activeStreams := make([]*pb.ActiveStream, 0)
	for _, stream := range r.streams {
		if stream.IsRecording || stream.IsStreaming {
			activeStreams = append(activeStreams, stream)
		}
	}
	return activeStreams
}
