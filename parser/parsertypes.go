package parser

// Further documentation on bros datatypes can be found on the bro website at:
// https://www.bro.org/sphinx/script-reference/types.html
// It is of value to note that many of theese types have applications specific
// to bro script and will likely never be implemented as types with any meaning
// in ai-hunt.
const (
	// BOOL reflects true or false, designated 'T' or 'F'
	BOOL = "bool"

	// COUNT is a numeric representation of a UINT_64 represented as either
	// a string of digits or a hex number. Note that hex numbers will begin
	// with the traditional 0x
	COUNT = "count"

	// INT is a numeric type representing an INT_64 represetned by a string
	// of digits preceded by either a '+' or a '-'. Note that INT may also
	// be expressed in hex and will maintian its leading sign ('-0xff')
	INT = "int"

	// DOUBLE is a numeric type representing a double-precision float.
	// Representation is a string of digits with an optional decimal point
	// as well as optional '+' or '-' proceding the number. The number may
	// also be optionally scaled with e notation. So 1234 123.4 -123.4
	// +1.234 and .003E-23 are examples of valid double types.
	DOUBLE = "double"

	// TIME is a temporal type representing an absolute time. Until found
	// otherwise it will be assumed that all time values are UNIX-NANO.
	TIME = "time"

	// INTERVAL is a temporal type representing relative time. An Interval
	// constant is represented by by a numeric constant followed by a time
	// unit which is one of usec, msec, sec, min, hr, or day. An 's' may
	// be appended to the unit so 3.5 min and 3.5mins represent the same
	// value. Finally an optional '-' negates an interval, denoting past
	// time. So -12 hr is read as "twelve hours in the past."
	INTERVAL = "interval"

	// STRING is a type used to hold character string values.
	STRING = "string"

	// PATTERN is a type used to represent regular expressions. Pattern
	// documentation can be found at
	// http://flex.sourceforge.net/manual/Patterns.html
	PATTERN = "pattern"

	// PORT is a type used to represent transport-level port numbers these
	// are typically represented as a number followed by one of /udp, /tcp,
	// /icmp, or /unkown.
	PORT = "port"

	// ADDR is a type used to represent an IP address. IPv4 addresses are
	// represented in dotted quad notation. IPv6 addresses are written in
	// colon hex notation as outlined in RFC 2373 (including the mixed
	// notation which allows dotted quad IPv4 addresses in the lower 32
	// bits) and further placed into brackets. So [::ffff:192.168.1.100]
	// can be used to represent the IPv4 address 192.168.1.100.
	ADDR = "addr"

	// SUBNET is a type used to represent a subnet in CIDR notation. So
	// 10.10.150.0/24 and [fe80::]/64 are valid subnets.
	SUBNET = "subnet"

	// ENUM is a type allowing the specification of a set of related
	// values that have no further structure.
	ENUM = "enum"

	// TABLE represents an associated array that maps from one set of
	// values to another. Values being mapped are refered to as indices and
	// the resulting map the yield.
	//TABLE = "table"

	// SET is like table but the collection of indicies do not have to map
	// to any yield value.
	//SET = "set"

	// VECTOR is a table which is always mapped by its count.
	//VECTOR = "vector"

	// RECORD represents a collection of values each with a field name and
	// type.
	//RECORD = "record"

	// STRING_SET is a SET which contains STRINGs
	STRING_SET = "set[string]"

	// ENUM_SET is a SET which contains ENUMs
	ENUM_SET = "set[enum]"

	// STRING_VECTOR is a VECTOR which contains STRINGs
	STRING_VECTOR = "vector[string]"

	// INTERVAL_VECTOR is a VECTOR which contains INTERVALs
	INTERVAL_VECTOR = "vector[interval]"

	// FUNCTION represents a function type in bro script.
	FUNCTION = "function"

	// EVENT represents an event handler in bro script.
	EVENT = "event"

	// HOOK represents a bro script object best described as as the an
	// intersection of a function and an event.
	HOOK = "hook"

	// A file object which can be written to, but not read from (which is a
	// limitation of bro script and has nothing to do with brosync).
	FILE = "file"

	// OPAQUE represents data whos type is intentionally hidden, but whose
	// values may be passed to certain bro script builtins.
	OPAQUE = "opaque"

	// ANY is used to bypass strong typing in bro script.
	ANY = "any"
)
