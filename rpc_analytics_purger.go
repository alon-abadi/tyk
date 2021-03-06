package main

import (
	"encoding/json"
	"time"

	"gopkg.in/vmihailenco/msgpack.v2"
)

// Purger is an interface that will define how the in-memory store will be purged
// of analytics data to prevent it growing too large
type Purger interface {
	PurgeCache()
	PurgeLoop(time.Duration)
}

// RPCPurger will purge analytics data into a Mongo database, requires that the Mongo DB string is specified
// in the Config object
type RPCPurger struct {
	Store     *RedisClusterStorageManager
}

// Connect Connects to RPC
func (r *RPCPurger) Connect() {
	if RPCClientIsConnected {
		if RPCCLientSingleton != nil {
			if RPCFuncClientSingleton != nil {
				log.Info("RPC Analytics client using singleton")
				return
			}
		}
	}

	return
}

// PurgeLoop starts the loop that will pull data out of the in-memory
// store and into RPC.
func (r RPCPurger) PurgeLoop(sleep time.Duration) {
	for {
		time.Sleep(sleep)
		r.PurgeCache()
	}
}

// PurgeCache will pull the data from the in-memory store and drop it into the specified MongoDB collection
func (r *RPCPurger) PurgeCache() {
	//var AnalyticsValues []interface{}

	AnalyticsValues := r.Store.GetAndDeleteSet(ANALYTICS_KEYNAME)

	if len(AnalyticsValues) > 0 {
		keys := make([]AnalyticsRecord, len(AnalyticsValues), len(AnalyticsValues))

		for i, v := range AnalyticsValues {
			decoded := AnalyticsRecord{}
			err := msgpack.Unmarshal(v.([]byte), &decoded)
			log.Debug("Decoded Record: ", decoded)
			if err != nil {
				log.Error("Couldn't unmarshal analytics data:")
				log.Error(err)
			} else {
				keys[i] = decoded
			}
		}

		data, dErr := json.Marshal(keys)
		if dErr != nil {
			log.Error("Failed to marshal analytics data")
			return
		}

		// Send keys to RPC
		_, callErr := RPCFuncClientSingleton.Call("PurgeAnalyticsData", string(data))
		if callErr != nil {
			log.Error("Failed to call purge: ", callErr)
		}
	}

}
