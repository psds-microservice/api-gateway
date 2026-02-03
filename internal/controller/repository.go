package controller

import (
	"sync"
	"time"

	pb "github.com/psds-microservice/api-gateway/pkg/gen"
)

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

func (r *ClientRepository) SaveClient(client *pb.ClientInfo) {
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

func (r *ClientRepository) GetClient(clientID string) *pb.ClientInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients[clientID]
}

func (r *ClientRepository) RemoveClient(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, clientID)
}

func (r *ClientRepository) GetAllClients() []*pb.ClientInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	clients := make([]*pb.ClientInfo, 0, len(r.clients))
	for _, client := range r.clients {
		clients = append(clients, client)
	}
	return clients
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

func (r *StreamRepository) SaveStream(streamID string, stream *pb.ActiveStream) {
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

func (r *StreamRepository) UpdateStats(streamID string, frame *pb.VideoFrame) *pb.StreamStats {
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

func (r *StreamRepository) GetStats(streamID string) *pb.StreamStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats[streamID]
}

func (r *StreamRepository) GetAllStats() []*pb.StreamStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	allStats := make([]*pb.StreamStats, 0, len(r.stats))
	for _, stats := range r.stats {
		allStats = append(allStats, stats)
	}
	return allStats
}

func (r *StreamRepository) GetStream(streamID string) *pb.ActiveStream {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.streams[streamID]
}

func (r *StreamRepository) GetAllStreams() []*pb.ActiveStream {
	r.mu.RLock()
	defer r.mu.RUnlock()
	streams := make([]*pb.ActiveStream, 0, len(r.streams))
	for _, stream := range r.streams {
		streams = append(streams, stream)
	}
	return streams
}

func (r *StreamRepository) RemoveStream(streamID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.streams, streamID)
	delete(r.stats, streamID)
}

func (r *StreamRepository) GetAllActiveStreams() []*pb.ActiveStream {
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
