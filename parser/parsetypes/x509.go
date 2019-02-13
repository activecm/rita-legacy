package parsetypes

import (
	"github.com/activecm/rita/config"
)

type (
	// x509 provides a data structure for bro's connection data
	x509 struct {
		// TimeStamp of this connection
		TimeStamp int64 `bson:"ts" bro:"ts" brotype:"time"`
		// FileID is the file id of this certificate.
		FileID string `bson:"file_id" bro:"id" brotype:"string"`
		// CertificateVersion : version number
		CertificateVersion int `bson:"cert_version" bro:"certificate.version" brotype:"count"`
		// CertificateSerial : serial number
		CertificateSerial string `bson:"cert_serial" bro:"certificate.serial" brotype:"string"`
		// CertificateSubject : subject
		CertificateSubject string `bson:"cert_subject" bro:"certificate.subject" brotype:"string"`
		// CertificateIssuer : issuer
		CertificateIssuer string `bson:"cert_issuer" bro:"certificate.issuer" brotype:"string"`
		// CommonName : last (most specific) common name
		CommonName string `bson:"common_name" bro:"cn" brotype:"string"`
		// CertNotValidBefore : Timestamp before when certificate is not valid.
		CertNotValidBefore int64 `bson:"cert_not_valid_before" bro:"certificate.not_valid_before" brotype:"time"`
		// CertNotValidAfter : Timestamp after when certificate is not valid
		CertNotValidAfter int64 `bson:"cert_not_valid_after" bro:"certificate.not_valid_after" brotype:"time"`
		// CertificateKeyAlg : Name of the key algorithm
		CertificateKeyAlg string `bson:"cert_key_alg" bro:"certificate.key_alg" brotype:"string"`
		// CertificateSigAlg : Name of the signature algorithm
		CertificateSigAlg string `bson:"cert_sig_alg" bro:"certificate.sig_alg" brotype:"string"`
		// CertificateKeyType : Key type, if key parseable by openssl (either rsa, dsa or ec)
		CertificateKeyType string `bson:"cert_key_type" bro:"certificate.key_type" brotype:"string"`
		// CertificateKeyLength : Key length in bits
		CertificateKeyLength int `bson:"cert_key_length" bro:"certificate.key_length" brotype:"count"`
		// CertificateExponent : Exponent, if RSA-certificate
		CertificateExponent string `bson:"cert_exponent" bro:"certificate.exponent" brotype:"string"`
		// CertificateCurve : Curve, if EC-certificate
		CertificateCurve string `bson:"cert_curve" bro:"certificate.curve" brotype:"string"`
		// SanDNS : List of DNS entries in SAN (subject alternative name)
		SanDNS []string `bson:"san_dns" bro:"san.dns" brotype:"vector[string]"`
		// SanURI : List of URI entries in SAN (subject alternative name)
		SanURI []string `bson:"san_uri" bro:"san.uri" brotype:"vector[string]"`
		// SanEmail : List of email entries in SAN (subject alternative name)
		SanEmail []string `bson:"san_email" bro:"san.email" brotype:"vector[string]"`
		// SanIP : List of IP entries in SAN (subject alternative name)
		SanIP []string `bson:"san_ip" bro:"san.ip" brotype:"vector[addr]"`
		// BasicConstraintsCA : CA flag set?
		BasicConstraintsCA bool `bson:"basic_constraints_ca" bro:"basic_constraints.ca" brotype:"bool"`
		// BasicConstraintsPathLen: Maximum path length
		BasicConstraintsPathLen bool `bson:"basic_constraints_path_len" bro:"basic_constraints.path_len" brotype:"count"`
	}
)

//TargetCollection returns the mongo collection this entry should be inserted into
func (in *x509) TargetCollection(config *config.StructureTableCfg) string {
	return config.X509Table
}

//Indices gives MongoDB indices that should be used with the collection
func (in *x509) Indices() []string {
	return []string{"$hashed:file_id"}
}
