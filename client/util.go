package client

import (
	"strings"

	"github.com/deejross/mydis/pb"
	"github.com/deejross/mydis/util"
)

func getPrefix(key string) []byte {
	bkey := util.StringToBytes(key)
	end := make([]byte, len(bkey))
	copy(end, bkey)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i]++
			end = end[:i+1]
			return end
		}
	}
	return util.ZeroByte
}

// GetKeyPrefix returns the actual rangeStart and rangeEnd for keys that end with '*'.
func GetKeyPrefix(key string) (bkey []byte, rangEnd []byte) {
	if strings.HasSuffix(key, "*") {
		newKey := strings.TrimSuffix(key, "*")
		rangEnd = getPrefix(newKey)
		bkey = util.StringToBytes(newKey)
		return
	}
	return util.StringToBytes(key), util.ZeroByte
}

// GetPermission returns a new Permission object for the given information or nil if permName unrecognized.
func GetPermission(key string, permName string) *pb.Permission {
	bkey, rangeEnd := GetKeyPrefix(key)
	permType, ok := pb.Permission_Type_value[strings.ToUpper(strings.TrimSpace(permName))]
	if !ok {
		return nil
	}

	return &pb.Permission{
		Key:      bkey,
		RangeEnd: rangeEnd,
		PermType: pb.Permission_Type(permType),
	}
}
