/*Package odfl provides tooling to connect to the Old Dominion API.  This is for truck shipments,
not small parcels.  Think LTL (less than truckload) shipments.  This code was created off the Ward API
documentation.  This uses and XML SOAP API.

Currently this package can perform:
- pickup requests

To create a pickup request:
- Set test or production mode (SetProductionMode()).
- Set shipper information (Shipper{}).
- Set shipment data (Consignee{}).
- Create the pickup request object (PickupRequest{}).
- Request the pickup (RequestPickup()).
- Check for any errors.
*/
package odfl

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

//odURL is the URL used to make API calls
var odURL = "http://www.odfl.com/wsPickup_v1b/services/ODPickupSOAP"

//timeout is the default time we should wait for a reply from Ward
//You may need to adjust this based on how slow connecting to Ward is for you.
//10 seconds is overly long, but sometimes Ward is very slow.
var timeout = time.Duration(10 * time.Second)

//testMode is used to make test or real pickup requests
//Set to false by SetProductionMode() to schedule real pickups
var testMode = true

//base XML data
var (
	soapenv = "http://schemas.xmlsoap.org/soap/envelope/"
	pic     = "http://pickup.ws.odfl.com"
)

//PickupRequest is the main body of the xml request
type PickupRequest struct {
	XMLName xml.Name `xml:"soapenv:Envelope"`

	SoapenvAttr string `xml:"xmlns:soapenv,attr"`
	PicAttr     string `xml:"xmlns:pic,attr"`

	Shipper   Shipper   `xml:"soapenv:Header>soapenv:Body>pic:pickupRequest>shipper"`
	Consignee Consignee `xml:"soapenv:Header>soapenv:Body>pic:pickupRequest>consignees>Consignee"`
}

//Shipper is the data on the shipper
type Shipper struct {
	//required
	ODFL4MeUser     string `xml:"odfl4meUser"` //web login
	ODFL4MePassword string `xml:"odfl4mePassword"`
	CompanyName     string `xml:"companyName"` //where shipment is coming from
	AddressLine1    string `xml:"addressLine1"`
	City            string `xml:"city"`
	StateProvince   string `xml:"stateProvince"` //two characters
	PostalCode      string `xml:"postalCode"`
	Country         string `xmml:"country"` //USA, CAN, or MEX
	ContactFName    string `xml:"contactFName"`
	ContactLName    string `xml:"contactLName"`
	PhoneAreaCode   string `xml:"phoneAreaCode"`  //first three of phone number, no + or +1
	PhoneNumber     string `xml:"phoneNumber"`    //last 7 digits of phone number
	TestFlag        bool   `xml:"testFlag"`       //set to true to NOT schedule a real pickup
	PickupDate      string `xml:"pickupDate"`     //yyyymmdd
	PickupTime      string `xml:"pickupTime"`     //hhmmss
	PickupTimeAMPM  string `xml:"pickupTimeAMPM"` //AM or PM
	WhoEntered      string `xml:"whoEntered"`     //who scheduled the pickup
	WhoPhoneNumber  string `xml:"whoPhoneNumber"`

	//optional
	AccountNumber string `xml:"accountNumber"` //odfl account number
	Attention     string `xml:"attention"`     //shipping dept or a person's name to contact with pickup issues
	AddressLine2  string `xml:"addressLine2"`
	PhoneExt      string `xml:"phoneExt"` //no "x" or non-numeric characters
	FaxAreaCode   string `xml:"faxAreaCode"`
	FaxNumber     string `xml:"faxNumber"`
	Email         string `xml:"email"`
	Comments      string `xml:"comments"`      //special instructions or special services
	DockCloseTime string `xml:"dockCloseTime"` //hhmmss
	DockCloseAMPM string `xml:"dockCloseAMPM"` //AM or PM
}

//Consignee is where the shipment is going and what the shipment is
type Consignee struct {
	//required
	CustomerShipmentID string  `xml:"customerShipmentId"` //a unique identifier for this shipment
	City               string  `xml:"city"`
	StateProvince      string  `xml:"stateProvince"` //two chars
	PostalCode         string  `xml:"postalCode"`
	Country            string  `xml:"country"`
	PhoneAreaCode      string  `xml:"phoneAreaCode"`
	PhoneNumber        string  `xml:"phoneNumber"`
	HandlingUnits      uint    `xml:"handlingUnits"`
	Pieces             uint    `xml:"pieces"`   //number of pieces, skids, etc.
	UnitType           string  `xml:"unitType"` //BDL, CRT, CTN, DRUM, SKID, OTH
	Weight             float64 `xml:"weight"`

	//optional
	PaymentMethod string `xml:"paymentMethod"` //P or C (prepaid or collect)
	CompanyName   string `xml:"companyName"`   //where shipment is delivering to
	AddressLine1  string `xml:"addressLine1"`
	AddressLine2  string `xml:"addressLine2"`
	ContactFName  string `xml:"contactFName"`
	ContactLName  string `xml:"contactLName"`
	PhoneExt      string `xml:"phoneExt"`
	FaxAreaCode   string `xml:"faxAreaCode"`
	FaxNumber     string `xml:"faxNumber"`
	Email         string `xml:"email"`
	Hazmat        string `xml:"hazmat"`
	Freezable     string `xml:"freezable"`
	Description   string `xml:"description"`
}

//SetProductionMode chooses the production url for use
func SetProductionMode(yes bool) {
	if yes {
		testMode = false
	}

	return
}

//SetTimeout updates the timeout value to something the user sets
//use this to increase the timeout if connecting to Ward is really slow
func SetTimeout(seconds time.Duration) {
	timeout = time.Duration(seconds * time.Second)
	return
}

//RequestPickup performs the call to the ODFL API to schedule a pickup
func (p *PickupRequest) RequestPickup() (responseData map[string]interface{}, err error) {
	//convert the pickup request to an xml
	xmlBytes, err := xml.Marshal(p)
	if err != nil {
		err = errors.Wrap(err, "odfl.RequestPickup - could not marshal xml")
		return
	}

	//add xml attributes
	p.SoapenvAttr = soapenv
	p.PicAttr = pic

	//set test mode
	p.Shipper.TestFlag = testMode

	//make the call to the ward API
	//set a timeout since golang doesn't set one by default and we don't want this to hang forever
	httpClient := http.Client{
		Timeout: timeout,
	}
	res, err := httpClient.Post(odURL, "text/xml", bytes.NewReader(xmlBytes))
	if err != nil {
		err = errors.Wrap(err, "odfl.RequestPickup - could not make post request")
		return
	}

	//read the response
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		err = errors.Wrap(err, "odfl.RequestPickup - could not read response 1")
		return
	}

	err = xml.Unmarshal(body, &responseData)
	if err != nil {
		err = errors.Wrap(err, "odfl.RequestPickup - could not read response 2")
		return
	}

	log.Println(responseData)

	//pickup request successful
	//response data will have confirmation info
	return
}
