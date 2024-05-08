package speedtest

import (
	"context"
	"github.com/showwin/speedtest-go/speedtest/transport"
	"net"
	"time"
)

type PacketLossAnalyzerOptions struct {
	RemoteSamplingInterval time.Duration
	SamplingDuration       time.Duration
	PacketSendingInterval  time.Duration
	PacketSendingTimeout   time.Duration
	SourceInterface        string      // source interface
	TCPDialer              *net.Dialer // tcp dialer for sampling
	UDPDialer              *net.Dialer // udp dialer for sending packet

}

type PacketLossAnalyzer struct {
	options *PacketLossAnalyzerOptions
}

func NewPacketLossAnalyzer(options *PacketLossAnalyzerOptions) (*PacketLossAnalyzer, error) {
	if options == nil {
		options = &PacketLossAnalyzerOptions{}
	}
	if options.SamplingDuration == 0 {
		options.SamplingDuration = time.Second * 30
	}
	if options.RemoteSamplingInterval == 0 {
		options.RemoteSamplingInterval = 1 * time.Second
	}
	if options.PacketSendingInterval == 0 {
		options.PacketSendingInterval = 67 * time.Millisecond
	}
	if options.PacketSendingTimeout == 0 {
		options.PacketSendingTimeout = 5 * time.Second
	}
	if options.TCPDialer == nil {
		options.TCPDialer = &net.Dialer{
			Timeout: options.PacketSendingTimeout,
		}
	}
	if options.UDPDialer == nil {
		var addr net.Addr
		if len(options.SourceInterface) > 0 {
			// skip error and using auto-select
			addr, _ = net.ResolveUDPAddr("udp", options.SourceInterface)
		}
		options.UDPDialer = &net.Dialer{
			Timeout:   options.PacketSendingTimeout,
			LocalAddr: addr,
		}
	}
	return &PacketLossAnalyzer{
		options: options,
	}, nil
}

func (pla *PacketLossAnalyzer) Run(host string, callback func(packetLoss *transport.PLoss)) error {
	ctx, cancel := context.WithTimeout(context.Background(), pla.options.SamplingDuration)
	defer cancel()
	return pla.RunWithContext(ctx, host, callback)
}

func (pla *PacketLossAnalyzer) RunWithContext(ctx context.Context, host string, callback func(packetLoss *transport.PLoss)) error {
	samplerClient, err := transport.NewClient(pla.options.TCPDialer)
	if err != nil {
		return transport.ErrUnsupported
	}
	senderClient, err := transport.NewPacketLossSender(samplerClient.ID(), pla.options.UDPDialer)
	if err != nil {
		return transport.ErrUnsupported
	}

	if err = samplerClient.Connect(ctx, host); err != nil {
		return transport.ErrUnsupported
	}
	if err = senderClient.Connect(ctx, host); err != nil {
		return transport.ErrUnsupported
	}
	if err = samplerClient.InitPacketLoss(); err != nil {
		return transport.ErrUnsupported
	}
	go pla.loopSender(ctx, senderClient)
	return pla.loopSampler(ctx, samplerClient, callback)
}

func (pla *PacketLossAnalyzer) loopSampler(ctx context.Context, client *transport.Client, callback func(packetLoss *transport.PLoss)) error {
	ticker := time.NewTicker(pla.options.RemoteSamplingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if pl, err1 := client.PacketLoss(); err1 == nil {
				if pl != nil {
					callback(pl)
				}
			} else {
				return err1
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (pla *PacketLossAnalyzer) loopSender(ctx context.Context, senderClient *transport.PacketLossSender) {
	order := 0
	sendTick := time.NewTicker(pla.options.PacketSendingInterval)
	defer sendTick.Stop()
	for {
		select {
		case <-sendTick.C:
			_ = senderClient.Send(order)
			order++
		case <-ctx.Done():
			return
		}
	}
}
