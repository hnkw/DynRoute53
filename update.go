package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

func chomp(s string) string {
	return strings.TrimRight(s, "\n")
}

func checkIP() (string, error) {
	resp, err := http.Get("http://checkip.amazonaws.com/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status not ok, status %d", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return chomp(string(b)), nil
}

func prepare() (*route53.Route53, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return route53.New(sess), nil
}

func zoneID(srv *route53.Route53, zoneName string) (string, error) {
	zones, err := srv.ListHostedZones(nil)
	if err != nil {
		return "", err
	}

	var zID string
	for _, z := range zones.HostedZones {
		if aws.StringValue(z.Name) == zoneName {
			zID = aws.StringValue(z.Id)
		}
	}
	if zID == "" {
		return "", fmt.Errorf("zone not found, %+v", zones)
	}

	return zID, nil
}

func fqdn(zoneName, hostName string) string {
	return strings.Join([]string{hostName, zoneName}, ".")
}

func ipAddressAvailable(srv *route53.Route53, zoneID, zoneName, hostName, currentIP string) (bool, error) {
	recordSets, err := srv.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId: &zoneID,
	})
	if err != nil {
		return false, err
	}

	f := fqdn(zoneName, hostName)
	for _, set := range recordSets.ResourceRecordSets {
		if aws.StringValue(set.Name) == f && aws.StringValue(set.Type) == route53.RRTypeA {
			for _, rs := range set.ResourceRecords {
				if aws.StringValue(rs.Value) == currentIP {
					return true, nil
				}
			}
		}
	}

	// current IP not found in DNS records
	return false, nil
}

func updateRecodeValue(srv *route53.Route53, zID, zoneName, hostName, ip string) error {
	f := fqdn(zoneName, hostName)
	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				&route53.Change{
					Action: aws.String(route53.ChangeActionUpsert),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(f),
						Type: aws.String(route53.RRTypeA),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(ip),
							},
						},
						TTL: aws.Int64(300),
					},
				},
			},
		},
		HostedZoneId: aws.String(zID),
	}
	_, err := srv.ChangeResourceRecordSets(params)
	return err
}

func update(zoneName, hostName string) error {
	currentIP, err := checkIP()
	if err != nil {
		return err
	}
	log.Printf("currentIP %s", currentIP)

	srv, err := prepare()
	if err != nil {
		return err
	}

	zID, err := zoneID(srv, zoneName)
	if err != nil {
		return err
	}

	ok, err := ipAddressAvailable(srv, zID, zoneName, hostName, currentIP)
	if err != nil {
		return err
	}
	if ok {
		log.Printf("ip address not changed")
		return nil
	}

	return updateRecodeValue(srv, zID, zoneName, hostName, currentIP)
}
