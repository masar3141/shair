package local

import (
	"context"
	"fmt"
	"os"

	"github.com/brutella/dnssd"
	"github.com/masar3141/shair"
)

const MDNSSERVICE = "_shair._tcp"

//	broadcasts an mDNS service record for the local Device, allowing other peers
//
// to discover the service and obtain the TCP server information to send data to.
// It advertises the service with the "_shair._tcp" service type on the specified port and the name of the host.
// This function runs until the context is cancelled by the caller, continuously broadcasting the service.
func (l *LocalShairer) broadcast(ctx context.Context, name string) {

	svCfg := dnssd.Config{
		Name:   name,
		Type:   MDNSSERVICE,
		Domain: "local",
		Port:   l.port,
	}

	sv, err := dnssd.NewService(svCfg)
	if err != nil {
		panic(fmt.Errorf("Mdns.Announce: Couldn't create the service: %w", err))
	}

	rp, err := dnssd.NewResponder()
	if err != nil {
		panic(fmt.Errorf("broadcast err responder: %w", err))
	}

	hdl, err := rp.Add(sv)
	_ = hdl
	if err != nil {
		return
	}

	go func() {
		<-ctx.Done()
		rp.Remove(hdl)
	}()

	err = rp.Respond(ctx)
	if err != nil {
		return
	}
}

// Discover continuously listens for mDNS service announcements from other nodes on the local network.
// It filters services matching "_shair._tcp" and sends notifications through a channel
// indicating whether a service was added or removed.
// This function runs indefinitely until the provided context is canceled.
func (l *LocalShairer) discover(
	ctx context.Context,
	peerCh chan<- shair.PeerUpdate,
) {
	addFn := func(e dnssd.BrowseEntry) {
		// TODO: figure out why it self discovers itself. for now, use that check to prevent self discovery
		hostname, _ := os.Hostname()
		if e.Name == hostname {
			return
		}

		dvc := &shair.Device{
			Name:         e.Name,
			DiscoveredOn: shair.Local,
			LocalInfo: shair.LocalInfo{
				IP:      e.IPs[0],
				SvcPort: e.Port,
			},
		}
		peerCh <- shair.PeerUpdate{Peer: dvc, Status: shair.Discovered}

		// store the device in the service to device map
		l.smu.Lock()
		defer l.smu.Unlock()
		l.serviceToDevice[e.Name] = dvc

		// store the tcp info in the device to tcp map
		l.dmu.Lock()
		defer l.dmu.Unlock()
		l.deviceToTCP[dvc] = tcpInfo{e.IPs[0], e.Port}
	}

	rmvFn := func(e dnssd.BrowseEntry) {
		// get the device associated with the service
		dvcToRmv, found := l.serviceToDevice[e.Name]
		if !found {
			// shouldn't reach there
			return
		}

		// send update through channel
		peerCh <- shair.PeerUpdate{Peer: dvcToRmv, Status: shair.Removed}

		// delete entries in the maps
		l.smu.Lock()
		defer l.smu.Unlock()
		delete(l.serviceToDevice, e.Name)

		l.dmu.Lock()
		defer l.dmu.Unlock()
		delete(l.deviceToTCP, dvcToRmv)
	}

	svc := fmt.Sprintf("%s.local.", MDNSSERVICE)
	dnssd.LookupType(ctx, svc, addFn, rmvFn)
}
