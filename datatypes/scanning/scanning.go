package scanning

type (
	Scan struct {
		ConnectionCount int     `bson:"connection_count"`
		Src             string  `bson:"src"`
		Dst             string  `bson:"dst"`
		LocalSrc        bool    `bson:"local_src"`
		LocalDst        bool    `bson:"local_dst"`
		TotalBytes      int     `bson:"total_bytes"`
		AverageBytes    float32 `bson:"average_bytes"`
		TotalDuration   float32 `bson:"total_duration"`
		PortSet         []int   `bson:"port_set"`
		PortCount       int     `bson:"port_count"`
	}
)
