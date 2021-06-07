//
// This part of the otcdns package offers a client that provides the ability to create and cleanup the needed TXT records in the OTC DNS.
// The methods are implemented in a way that they are most useful to build a solver. This is not a generic library.
//
package otcdns

import (
	"fmt"

	otc "github.com/opentelekomcloud/gophertelekomcloud"
	otcos "github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/dns/v2/recordsets"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/dns/v2/zones"
)

const (
	dnsRecordTypeTxt     string = "TXT"
	dnsRecordDescription string = "ACME Challenge"
	acmeChallengePrefix  string = "_acme-challenge."
)

//
// The DNS client we use to trigger our DNS actions.
//
type OtcDnsClient struct {
	Sc *otc.ServiceClient

	//
	// Optional subdomain, which will be inserted between "_acme-challenge." and the zone name.
	//
	Subdomain string
}

//
// Creates a new DNSv2 ServiceClient.
// See also gophertelekomcloud/acceptance/clients/clients.go
//
func NewDNSV2ClientWithAuth(authOpts otc.AuthOptionsProvider, endpointOpts otc.EndpointOpts) (*OtcDnsClient, error) {

	providerClient, err := getProviderClientWithAccessKeyAuth(authOpts)
	if err != nil {
		return nil, fmt.Errorf("cannot create providerClient. %s", err)
	}

	serviceClient, err := otcos.NewDNSV2(providerClient, endpointOpts)
	if err != nil {
		return nil, fmt.Errorf("cannot create serviceClient. %s", err)
	}

	return &OtcDnsClient{Sc: serviceClient}, nil
}

//
// Creates a new DNSv2 ServiceClient.
// See also gophertelekomcloud/acceptance/clients/clients.go
//
func NewDNSV2Client() (*OtcDnsClient, error) {
	cloudsConfig, err := getCloud()
	if err != nil {
		return nil, err
	}
	endpointOpts := otc.EndpointOpts{
		Region: cloudsConfig.RegionName,
	}

	providerClient, err := getProviderClient()
	if err != nil {
		return nil, err
	}

	serviceClient, err := otcos.NewDNSV2(providerClient, endpointOpts)
	if err != nil {
		return nil, err
	}

	return &OtcDnsClient{Sc: serviceClient}, nil
}

// ===========================================================================
// Zones
// ===========================================================================

//
// Retrieves a Zone data structure by its name.
// https://pkg.go.dev/github.com/opentelekomcloud/gophertelekomcloud@v0.3.2/openstack/dns/v2/zones
// github.com/opentelekomcloud/gophertelekomcloud/openstack/dns/v2/zones
//
func (dnsClient *OtcDnsClient) GetHostedZone(zoneName string) (*zones.Zone, error) {

	listOpts := zones.ListOpts{
		Name: zoneName,
	}

	allPages, err := zones.List(dnsClient.Sc, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("zone %s not found: %s", zoneName, err)
	}

	allZones, err := zones.ExtractZones(allPages)
	if err != nil {
		return nil, fmt.Errorf("zone %s extraction failed: %s", zoneName, err)
	}

	// Debug
	//for _, zone := range allZones {
	//	fmt.Printf("%+v\n", zone)
	//}

	// We need exactly 1 zone to operate on
	if len(allZones) != 1 {
		return nil, fmt.Errorf("zone query with %s returned %d zones. Expected: 1", zoneName, len(allZones))
	}

	return &allZones[0], nil
}

// ===========================================================================
// RecordSets
// ===========================================================================

//
// Creates a new TXT recordset for the ACME challenge and sets the given challengeValue as TXT record.
// https://pkg.go.dev/github.com/opentelekomcloud/gophertelekomcloud@v0.3.2/openstack/dns/v2/recordsets
// github.com/opentelekomcloud/gophertelekomcloud/openstack/dns/v2/recordsets
//
func (dnsClient *OtcDnsClient) NewTxtRecordSet(zone *zones.Zone, challengeValue string) (*recordsets.RecordSet, error) {
	dnsName := dnsClient.getDnsName(zone.Name)
	createOpts := recordsets.CreateOpts{
		Name:        dnsName,
		Type:        dnsRecordTypeTxt,
		TTL:         300,
		Description: dnsRecordDescription,
		Records:     []string{challengeValue},
	}
	var pCreatedRecordset *recordsets.RecordSet
	pCreatedRecordset, err := recordsets.Create(dnsClient.Sc, zone.ID, createOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("create TXT record failed for %s: %s", challengeValue, err)
	}

	return pCreatedRecordset, nil
}

//
// Reads the TXT recordset created for the ACME challenge.
// Valid results are 1 or 0 recordsets.
// Error if query is not successful or more than 1 result.
//
func (dnsClient *OtcDnsClient) GetTxtRecordSet(zone *zones.Zone) (*recordsets.RecordSet, error) {
	dnsName := dnsClient.getDnsName(zone.Name)
	listOpts := recordsets.ListOpts{
		Type: dnsRecordTypeTxt,
		Name: dnsName,
	}

	allPages, err := recordsets.ListByZone(dnsClient.Sc, zone.ID, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("list records failed for dns entry %s: %s", dnsName, err)
	}

	allRRs, err := recordsets.ExtractRecordSets(allPages)
	if err != nil {
		return nil, fmt.Errorf("extract recordset failed for dns entry %s: %s", dnsName, err)
	}

	// Debug
	//for _, rr := range allRRs {
	//	fmt.Printf("%+v\n", rr)
	//}

	if len(allRRs) == 1 {
		// We need exactly 1 recordset to operate on
		return &allRRs[0], nil
	} else if len(allRRs) == 0 {
		// Query was successful, but no results
		return nil, nil
	} else {
		// More than 1 result.
		return nil, fmt.Errorf("query with %s returned %d recordsets. Expected: 1", dnsName, len(allRRs))
	}
}

//
// Tests, if a TXT recordset exists for the ACME challenge.
//
func (dnsClient *OtcDnsClient) HasTxtRecordSet(zone *zones.Zone) (bool, error) {
	dnsName := dnsClient.getDnsName(zone.Name)
	listOpts := recordsets.ListOpts{
		Type: dnsRecordTypeTxt,
		Name: dnsName,
	}

	allPages, err := recordsets.ListByZone(dnsClient.Sc, zone.ID, listOpts).AllPages()
	if err != nil {
		return false, fmt.Errorf("list records failed for dns name %s: %s", dnsName, err)
	}

	allRRs, err := recordsets.ExtractRecordSets(allPages)
	if err != nil {
		return false, fmt.Errorf("extract recordset failed for dns name %s: %s", dnsName, err)
	}

	// Debug
	//for _, rr := range allRRs {
	//	fmt.Printf("%+v\n", rr)
	//}

	if len(allRRs) == 1 {
		// Queries were successful, 1 entry found.
		return true, nil
	} else if len(allRRs) == 0 {
		// Queries were successful, but no entries found.
		return false, nil
	} else {
		// More than 1 result.
		return false, fmt.Errorf("query with %s returned %d recordsets. Expected: 1", dnsName, len(allRRs))
	}
}

//
// Deletes the given recordset. The intention is that the given zone and recordset are the ones
// created for the ACME challenge.
//
func (dnsClient *OtcDnsClient) DeleteRecordSet(zone *zones.Zone, recordset *recordsets.RecordSet) error {
	err := recordsets.Delete(dnsClient.Sc, zone.ID, recordset.ID).ExtractErr()
	if err != nil {
		return fmt.Errorf("deletion of record with zoneId %s and recordsetId %s failed: %s", zone.ID, recordset.ID, err)
	}

	return nil
}

// ===========================================================================
// Records in the Recordsets
// ===========================================================================

//
// Tests, if the given challengeValue exists in the TXT records of the recordset.
//
func (dnsClient *OtcDnsClient) HasTxtRecordValue(zone *zones.Zone, challengeValue string) (bool, *recordsets.RecordSet, error) {
	recordSet, err := dnsClient.GetTxtRecordSet(zone)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get recordset. %s", err)
	}

	if recordSet == nil {
		return false, nil, nil
	}

	var found bool = false
	// The recordset is present. Check key value.
	for _, keyRecord := range recordSet.Records {
		if challengeValue == keyRecord {
			found = true
			break
		}
	}

	if found {
		return true, recordSet, nil
	} else {
		return false, recordSet, nil
	}
}

//
// Updates the given recordset with the set of TXT records for the ACME challenge.
// This allows you to add or remove TXT value records.
//
// The challengeValues must have at least one entry. The OTC API has a bug. When we send an empty array the values are not deleted as expected.
//
func (dnsClient *OtcDnsClient) UpdateTxtRecordValues(zone *zones.Zone, recordset *recordsets.RecordSet, challengeValues []string) (*recordsets.RecordSet, error) {
	if len(challengeValues) == 0 {
		return nil, fmt.Errorf("update TXT records failed. The challengeValue records must have at least one entry")
	}
	updateOpts := recordsets.UpdateOpts{
		Records: challengeValues,
	}
	var pUpdatedRecordSet *recordsets.RecordSet
	var err error
	pUpdatedRecordSet, err = recordsets.Update(dnsClient.Sc, zone.ID, recordset.ID, updateOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("update TXT records failed for recordset ID %s: %s", recordset.ID, err)
	}

	return pUpdatedRecordSet, nil
}

//
// Deletes the given TXT value from the records for the ACME challenge.
//
// challengeValue: The value that shall be deleted.
// deleteRecordsetIfEmpty: The OTC API does not allow to delete the last TXT value.
//     If this is set to true, the whole recordset is deleted, when there value to delete is the last one.
//
func (dnsClient *OtcDnsClient) DeleteTxtRecordValue(zone *zones.Zone, challengeValue string, deleteRecordsetIfEmpty bool) (*recordsets.RecordSet, error) {
	challengeValueExists, existingRecordset, err := dnsClient.HasTxtRecordValue(zone, challengeValue)
	if err != nil {
		return nil, fmt.Errorf("failed to check existence of DNS TXT entry. %s", err)
	}
	if challengeValueExists {
		// The recordset is present. Check key value.
		var deleteIndex int = -1
		for currentIndex, keyRecord := range existingRecordset.Records {
			if challengeValue == keyRecord {
				deleteIndex = currentIndex
				break
			}
		}

		if deleteIndex >= 0 {
			changedRecords := append(existingRecordset.Records[:deleteIndex], existingRecordset.Records[deleteIndex+1:]...)
			if len(changedRecords) == 0 {
				if deleteRecordsetIfEmpty {
					err := dnsClient.DeleteRecordSet(zone, existingRecordset)
					if err != nil {
						return nil, fmt.Errorf("failed to delete recordset. %s", err)
					}
					return nil, nil
				} else {
					return nil, fmt.Errorf("failed to delete record value. Deletion of the last value is not possible. You can set deleteRecordsetIfEmpty to true, to delete the whole recordset in this case")
				}
			} else {
				var pChangedRecordset *recordsets.RecordSet
				var err error
				pChangedRecordset, err = dnsClient.UpdateTxtRecordValues(zone, existingRecordset, changedRecords)
				if err != nil {
					return nil, fmt.Errorf("failed to update DNS TXT entry with deleted record. %s", err)
				}
				return pChangedRecordset, nil
			}
		} else {
			// Value not found
			return nil, fmt.Errorf("failed to delete record value. Value not found")
		}
	} else if existingRecordset == nil {
		return nil, fmt.Errorf("failed to delete record value. Recordset not found")
	} else {
		return nil, fmt.Errorf("failed to delete record value. Value not found")
	}
}

//
// Ensures that a valid subdomain part is set.
//
func (dnsClient *OtcDnsClient) getDnsName(zoneName string) string {
	if dnsClient.Subdomain == "" {
		dnsName := acmeChallengePrefix + zoneName
		return dnsName
	} else {
		dnsName := dnsClient.Subdomain + "." + zoneName
		return dnsName
	}
}
