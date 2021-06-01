// The tests in this file test the ability to utilize the otc dns api.
// - Creating a otc dns client.
// - Reading zone information
// - Reading dns records
// - Creating and deleting a TXT record.
//
// TODO: The tests are currently based on "no error = test pass", but in some cases test assertions would make more sense.
//       E.g. check that the result in TestHasTxtRecordSet is true, not only that the method throws no error.
//
// VSCode users: Add { ... "go.testFlags": ["-v"] ... } to your settings.json to view log output for non failing tests.
//
// Be aware that some DNS operations may have a delay.
//
package otcdns

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	// otctools "github.com/opentelekomcloud/gophertelekomcloud/acceptance/tools"

	otc "github.com/opentelekomcloud/gophertelekomcloud"
	otctools "github.com/opentelekomcloud/gophertelekomcloud/acceptance/tools"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/dns/v2/recordsets"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/dns/v2/zones"
	"github.com/stretchr/testify/assert"
)

// TODO: Move to test config file
const (
	testSubdomain = "cert-manager-dns01-tests"
	sleepTime     = 2 * time.Second
)

//
// Allows to overwrite the default value with a custom value.
//
func getTestZone() string {
	var testZone string = "hpi-schul-cloud.dev."
	if os.Getenv("TEST_ZONE_NAME") == "" {
		return testZone
	} else {
		return os.Getenv("TEST_ZONE_NAME")
	}
}

// ===========================================================================
// Client and Zones
// ===========================================================================

//
// Tests, if we can create a otcdns client and if we can retrieve at least
// one zone record.
//
func TestClientCreateWithUser(t *testing.T) {
	t.Log("TestClientCreateWithUser start")

	// We load the cloud config here, to circumvent that we store duplicated configuration data only.
	// The testcase will be independent from the environment and clouds.yaml configuration.
	cloudsConfig, err := getCloudProfile(OtcProfileNameUser)
	if err != nil {
		log.Fatalf("Test preconditions not met. A local clouds.yaml must exist: %s", err)
	}

	// Prepare our test parameters from cloud config.
	authOpts := otc.AuthOptions{
		IdentityEndpoint: cloudsConfig.AuthInfo.AuthURL,
		Username:         cloudsConfig.AuthInfo.Username,
		Password:         cloudsConfig.AuthInfo.Password,
		DomainName:       cloudsConfig.AuthInfo.DomainName,
		TenantID:         cloudsConfig.AuthInfo.ProjectID,
	}

	endpointOpts := otc.EndpointOpts{
		Region: cloudsConfig.RegionName,
	}

	// Now we can start the test
	// Create a client
	client, err := NewDNSV2ClientWithAuth(authOpts, endpointOpts)
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	runZoneTest(t, client)

	t.Log("TestClientCreateWithUser end")
}

//
// Tests, if we can create a otcdns client and if we can retrieve at least
// one zone record.
//
func TestClientCreateWithAkSk(t *testing.T) {
	t.Log("TestClientCreateWithTokenAuth start")

	// We load the cloud config here, to circumvent that we store duplicated configuration data only.
	// The testcase will be independent from the environment and clouds.yaml configuration.
	cloudsConfig, err := getCloudProfile(OtcProfileNameAkSk)
	if err != nil {
		log.Fatalf("Test preconditions not met. A local clouds.yaml must exist: %s", err)
	}

	// Prepare our test parameters from cloud config.
	authOpts := otc.AKSKAuthOptions{
		IdentityEndpoint: cloudsConfig.AuthInfo.AuthURL,
		AccessKey:        cloudsConfig.AuthInfo.AccessKey,
		SecretKey:        cloudsConfig.AuthInfo.SecretKey,
	}

	endpointOpts := otc.EndpointOpts{
		Region: cloudsConfig.RegionName,
	}

	// Now we can start the test
	// Create a client
	client, err := NewDNSV2ClientWithAuth(authOpts, endpointOpts)
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	runZoneTest(t, client)

	t.Log("TestClientCreateWithTokenAuth end")
}

//
// Tests, if we can create a otcdns client and if we can retrieve at least
// one zone record.
//
// In this case we utilize the configuration that is given by the environment / clouds.yaml.
// We can directly start the test.
//
func TestClientCreateWithCloudConfig(t *testing.T) {
	t.Log("TestClientCreateWithZoneList start")

	// Create a client
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	runZoneTest(t, client)

	t.Log("TestClientCreateWithZoneList end")
}

func runZoneTest(t *testing.T, client *OtcDnsClient) {
	client.Subdomain = testSubdomain

	var allZones []zones.Zone
	allPages, err := zones.List(client.Sc, nil).AllPages()
	if err != nil {
		t.Fatalf("Unable to retrieve zones: %v", err)
	}

	allZones, err = zones.ExtractZones(allPages)
	if err != nil {
		t.Fatalf("Unable to extract zones: %v", err)
	}

	// Details for debugging
	//for _, zone := range allZones {
	//	otctools.PrintResource(t, &zone)
	//}
	t.Log(fmt.Sprintf("Number of zones in %s: %d", getTestZone(), len(allZones)))

	// Test, if we have received more than 0 zone records.
	assert.Greater(t, len(allZones), 0, "There should be more than one zone entry for the testdomain.")
}

//
//
//
func TestGetDevZone(t *testing.T) {
	t.Log("TestGetDevZone start")
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	client.Subdomain = testSubdomain

	var pZone *zones.Zone
	pZone, err = client.GetHostedZone(getTestZone())
	if err != nil {
		t.Fatalf("Unable to extract zones: %v", err)
	}
	// Details for debugging
	// otctools.PrintResource(t, pZone)
	assert.Equal(t, getTestZone(), pZone.Name, "The queried zone name must be the name of the test zone.")
	t.Log("TestGetDevZone end")
}

// ===========================================================================
// RecordSets 1/2
// ===========================================================================

//
// Deletes the whole recordset.
//
// If needed a recordset to delete is created by the test.
//
func TestDeleteTxtRecordSetOnly(t *testing.T) {
	t.Log("TestDeleteTxtRecordSetOnly start")
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	client.Subdomain = testSubdomain

	pZone, err := client.GetHostedZone(getTestZone())
	if err != nil {
		t.Fatalf("Unable to get zone entry: %s", err)
	}

	testRecordset, err := client.GetTxtRecordSet(pZone)
	if err != nil {
		t.Fatalf("Unable to get TXT entry: %s", err)
	}

	if testRecordset != nil {
		// otctools.PrintResource(t, testRecordset)
		err = client.DeleteRecordSet(pZone, testRecordset)
		if err != nil {
			t.Fatalf("Unable to delete recordset: %s", err)
		}
	}

	t.Log("TestDeleteTxtRecordSetOnly end")
}

//
// Before we start our testsuite. Check, that the test record does not exist.
//
func TestHasTxtRecordSetMustNotExist(t *testing.T) {
	t.Log("TestHasTxtRecordSetMustNotExist start")
	t.Log("It is expected that this is the first test to run and that the test record is not created yet.")
	t.Log("If this fails, try running TestDeleteTxtRecordSet.")
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	client.Subdomain = testSubdomain

	pZone, err := client.GetHostedZone(getTestZone())
	if err != nil {
		t.Fatalf("Unable to get zone entry: %s", err)
	}

	recordSetExists, err := client.HasTxtRecordSet(pZone)
	if err != nil {
		t.Fatalf("Unable to get TXT entry: %s", err)
	}
	// otctools.PrintResource(t, recordSetExists)
	assert.Equal(t, false, recordSetExists, "The record must not exist.")

	t.Log("TestHasTxtRecordSetMustNotExist end")
}

//
// With this test we create our test record, which will be queried and manipulated in the following tests.
//
func TestNewTxtRecordSet(t *testing.T) {
	t.Log("TestNewTxtRecordSet start")
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	client.Subdomain = testSubdomain

	pZone, err := client.GetHostedZone(getTestZone())
	if err != nil {
		t.Fatalf("Unable to extract zones: %v", err)
	}

	txtValue := fmt.Sprintf("\"challenge test value %d\"", time.Now().UnixNano())
	pCreatedRecordset, err := client.NewTxtRecordSet(pZone, txtValue)
	if err != nil {
		t.Fatalf("Unable to create TXT entry: %s", err)
	}
	// Details for debugging
	// otctools.PrintResource(t, pCreatedRecordset)
	assert.Equal(t, txtValue, pCreatedRecordset.Records[0], "The queried record entry must match the desired value.")
	t.Log("TestNewTxtRecordSet end")
}

//
// Query the test record, created in the TestNewTxtRecordSet test.
//
func TestGetTxtRecordSet(t *testing.T) {
	t.Log("TestGetTxtRecordSet start")
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	client.Subdomain = testSubdomain

	pZone, err := client.GetHostedZone(getTestZone())
	if err != nil {
		t.Fatalf("Unable to get zone entry: %s", err)
	}

	var pRecordset *recordsets.RecordSet
	pRecordset, err = client.GetTxtRecordSet(pZone)
	if err != nil {
		t.Fatalf("Unable to get TXT entry: %s", err)
	}
	time.Sleep(sleepTime)
	// Details for debugging
	//otctools.PrintResource(t, pRecordset)
	assert.Equal(t, getTestZone(), pRecordset.ZoneName, "The queried zone name must match the test zone.")
	assert.Equal(t, "TXT", pRecordset.Type, "The queried record type must match TXT.")
	assert.Equal(t, "ACTIVE", pRecordset.Status, "The queried record status must match ACTIVE not PENDING or PENDING_CREATE.")

	t.Log("TestGetTxtRecordSet end")
}

//
// Exists test for the test record, created in the TestNewTxtRecordSet test.
//
func TestHasTxtRecordSet(t *testing.T) {
	t.Log("TestHasTxtRecordSet start")
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	client.Subdomain = testSubdomain

	pZone, err := client.GetHostedZone(getTestZone())
	if err != nil {
		t.Fatalf("Unable to get zone entry: %s", err)
	}

	recordSetExists, err := client.HasTxtRecordSet(pZone)
	if err != nil {
		t.Fatalf("Unable to get TXT entry: %s", err)
	}
	// otctools.PrintResource(t, recordSetExists)
	assert.Equal(t, true, recordSetExists, "The record must exist.")

	t.Log("TestHasTxtRecordSet end")
}

// ===========================================================================
// Records in the Recordsets
// ===========================================================================

//
// Appends a value record to the already existing one.
// After this we have a TXT entry with two value records.
//
func TestUpdateTxtRecordSetAddValue(t *testing.T) {
	t.Log("TestUpdateTxtRecordSetAddValue start")
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	client.Subdomain = testSubdomain

	zone, err := client.GetHostedZone(getTestZone())
	if err != nil {
		t.Fatalf("Unable to get zone entry: %s", err)
	}

	existingRecordset, err := client.GetTxtRecordSet(zone)
	if err != nil {
		t.Fatalf("Unable to get TXT entry: %s", err)
	}
	// otctools.PrintResource(t, existingRecordset.Records)
	if len(existingRecordset.Records) != 1 {
		t.Fatalf("There must be one value records. Test preconditions not given. %d", len(existingRecordset.Records))
	}

	txtValue := fmt.Sprintf("\"challenge test value %d\"", time.Now().UnixNano())
	changedRecords := append(existingRecordset.Records, txtValue)
	changedRecordset, err := client.UpdateTxtRecordValues(zone, existingRecordset, changedRecords)
	if err != nil {
		t.Fatalf("Unable to update TXT entry: %s", err)
	}
	// otctools.PrintResource(t, changedRecordset)
	assert.Equal(t, 2, len(changedRecordset.Records), "There must be two value records. One was added.")
	t.Log("TestUpdateTxtRecordSetAddValue end")
}

//
//
//
func TestDeleteTxtRecordValue(t *testing.T) {
	t.Log("TestDeleteTxtRecordValue start")

	time.Sleep(sleepTime)
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	client.Subdomain = testSubdomain

	zone, err := client.GetHostedZone(getTestZone())
	if err != nil {
		t.Fatalf("Unable to get zone entry: %s", err)
	}

	existingRecordset, err := client.GetTxtRecordSet(zone)
	if err != nil {
		t.Fatalf("Unable to get TXT entry: %s", err)
	}
	var originalNumberOfRecords int = len(existingRecordset.Records)
	if originalNumberOfRecords < 1 {
		t.Fatalf("There must be at least one value record. Test preconditions not given. Run TestUpdateTxtRecordSetAddValue to add one value. %d", originalNumberOfRecords)
	}

	deleteValue := existingRecordset.Records[0]
	t.Log(fmt.Sprintf("Deleting value: %s", deleteValue))
	changedRecordset, err := client.DeleteTxtRecordValue(zone, deleteValue, false)
	if err != nil {
		t.Fatalf("Unable to update TXT entry: %s", err)
	}
	var changedNumberOfRecords int = len(changedRecordset.Records)
	t.Log(fmt.Sprintf("Original number of records: %d, Changed number of records: %d", originalNumberOfRecords, changedNumberOfRecords))

	// otctools.PrintResource(t, changedRecordset)
	assert.Equal(t, originalNumberOfRecords, changedNumberOfRecords+1, "There must be one record less after value deletion.")

	t.Log("TestDeleteTxtRecordValue end")
}

//
//
//
func TestDeleteTxtRecordValueLastOne(t *testing.T) {
	t.Log("TestDeleteTxtRecordValueLastOne start")
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	client.Subdomain = testSubdomain

	zone, err := client.GetHostedZone(getTestZone())
	if err != nil {
		t.Fatalf("Unable to get zone entry: %s", err)
	}

	existingRecordset, err := client.GetTxtRecordSet(zone)
	if err != nil {
		t.Fatalf("Unable to get TXT entry: %s", err)
	}

	if existingRecordset == nil {
		t.Fatalf("There must be a test record created. Test preconditions not given. Call TestNewTxtRecordSet to create one.")
	}

	var originalNumberOfRecords int = len(existingRecordset.Records)
	if originalNumberOfRecords != 1 {
		t.Fatalf("There must be at exactly one value record. Test preconditions not given. %d", originalNumberOfRecords)
	}

	deleteValue := existingRecordset.Records[0]
	t.Log(fmt.Sprintf("Deleting value: %s. Record count: %d", deleteValue, len(existingRecordset.Records)))
	{
		var pChangedRecordset *recordsets.RecordSet = nil
		var err error
		pChangedRecordset, err = client.DeleteTxtRecordValue(zone, deleteValue, true)
		if err != nil {
			t.Fatalf("Unable to update TXT entry: %s", err)
		}

		// Must print "tools.go:<line>: null"
		otctools.PrintResource(t, pChangedRecordset)

		var isNil bool = (pChangedRecordset == nil)
		assert.Equal(t, true, isNil, "The returned recordset must nil1.")

		// I don't know why this does not work:
		// assert.Equal(t, true, pChangedRecordset, "The returned recordset must nil1.")
		// It reports:
		// Error:    Not equal:
		// expected: <nil>(<nil>)
		// actual  : *recordsets.RecordSet((*recordsets.RecordSet)(nil))
	}

	t.Log("TestDeleteTxtRecordValueLastOne end")
}

// ===========================================================================
// RecordSets 2/2
// ===========================================================================

//
// Deletes the whole recordset.
//
// If needed a recordset to delete is created by the test.
//
func TestCreateGetDeleteTxtRecordSet(t *testing.T) {
	t.Log("TestCreateGetDeleteTxtRecordSet start")
	client, err := NewDNSV2Client()
	if err != nil {
		t.Fatalf("Unable to create a DNS client: %v", err)
	}
	client.Subdomain = testSubdomain

	pZone, err := client.GetHostedZone(getTestZone())
	if err != nil {
		t.Fatalf("Unable to get zone entry: %s", err)
	}

	testRecordsetExist, err := client.HasTxtRecordSet(pZone)
	if err != nil {
		t.Fatalf("Unable to check existence of test recordset: %s", err)
	}

	var testRecordset *recordsets.RecordSet
	if testRecordsetExist {
		testRecordset, err = client.GetTxtRecordSet(pZone)
		if err != nil {
			t.Fatalf("Unable to get TXT entry: %s", err)
		}
	} else {
		txtValue := fmt.Sprintf("\"challenge test value %d\"", time.Now().UnixNano())
		testRecordset, err = client.NewTxtRecordSet(pZone, txtValue)
		if err != nil {
			t.Fatalf("Unable to create TXT entry: %s", err)
		}
	}

	// otctools.PrintResource(t, pRecordset)
	err = client.DeleteRecordSet(pZone, testRecordset)
	if err != nil {
		t.Fatalf("Unable to delete recordset: %s", err)
	}

	t.Log("TestCreateGetDeleteTxtRecordSet end")
}
