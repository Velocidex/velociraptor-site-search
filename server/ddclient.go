package server

import (
	"context"
	"net"
	"time"

	"github.com/Velocidex/velociraptor-site-search/api"
)

func (self *CloudflareUpdater) StartDDClientService(ctx context.Context) {

	logger := self.config_obj.GetLogger()
	logger.Info("Starting the DynDNS service: Updating hostname %v with checkip URL %v",
		self.config_obj.Hostname, self.external_ip_url)

	min_update_wait := self.config_obj.DynDns.Frequency
	if min_update_wait == 0 {
		min_update_wait = 60
	}

	// First time check immediately.
	self.updateIP(ctx, self.config_obj)

	for {
		select {
		case <-ctx.Done():
			return

			// Do not try to update sooner than this or we
			// get banned. It takes a while for dns
			// records to propagate.
		case <-time.After(time.Duration(min_update_wait) * time.Second):
			self.updateIP(ctx, self.config_obj)
		}
	}
}

func (self *CloudflareUpdater) GetCurrentDDNSIp(fqdn string) ([]string, error) {
	r := net.Resolver{
		PreferGo: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ips, err := r.LookupHost(ctx, fqdn)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

// Failing to update the DNS is not a fatal error we can try again
// later.
func (self *CloudflareUpdater) updateIP(
	ctx context.Context, config_obj *api.Config) {
	logger := config_obj.GetLogger()
	logger.Info("DynDns: Checking DNS with %v", self.external_ip_url)

	externalIP, err := self.GetExternalIp()
	if err != nil {
		logger.Error("DynDns: Unable to get external IP: %v", err)
		return
	}

	ddns_hostname := self.hostname

	// If we can not resolve the current hostname then lets try to
	// update it anyway.
	hostnameIPs, _ := self.GetCurrentDDNSIp(ddns_hostname)
	for _, ip := range hostnameIPs {
		if ip == externalIP {
			logger.Info("DynDns: Current IP is good %v", externalIP)
			return
		}
	}

	logger.Info("DynDns: DNS UPDATE REQUIRED. External IP=%v. %v=%v.",
		externalIP, ddns_hostname, hostnameIPs)

	err = self.UpdateDDNSRecord(ctx, config_obj, externalIP)
	if err != nil {
		logger.Error("DynDns: Unable to set dns: %v", err)
		return
	}
}
