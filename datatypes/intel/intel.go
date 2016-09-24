package data

import "time"

type (
	// IntelData provides a data abstraction for intel records
	// NOTE: Some of the fields in this document currently do not get filled
	//       in. They are here for future use.
	IntelData struct {
		// Hostname is the string host name if it has one
		HostName string `bson:"host_name"`

		// IP is the ip address of this host (indexed)
		IP string `bson:"ip"`

		// ASN provides the Autonomous Systems Number
		ASN int64 `bson:"asn"`

		// Prefix provides the BGP prefix of this hosts network
		Prefix string `bson:"bgp_prefix"`

		// CountryCode provides the country code for this record
		CountryCode string `bson:"country_code"`

		// Registry is the registry from which cymru got this record
		Registry string `bson:"registry"`

		// Allocated is the date this was allocated
		Allocated time.Time `bson:"allocated"`

		// Info refers to the info data from cymru
		Info string `bson:"cymru_info"`

		// ASName is the name string associated with the ASN
		ASName string `bson:"asn_name"`

		// IntelDate is the date this item was looked up by the intel module
		IntelDate time.Time `bson:"intel_date"`
	}
)
