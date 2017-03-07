Mydis
=====

Version: 0.5.0

Introduction
------------
Distributed, reliable cache library, server, and client. Inspired by Redis, this cache is written entirely in Go and can be used as a library, embedded into an existing application, or as a standalone client/server.

Basics
------
The cache can store multiple types of data: strings, bytes, integers, floats, lists, and hashes (objects that hold key/value pairs). Each item is referenced with a key, a string of any length.
The cache library, server, and client are thread/goroutine-safe. Client and server communication is handled with gRPC. All data types can have an expiration value set.
Both client and peer connections are encrypted by default.

Details
-------
Under the hood, the production-ready [Etcd](https://etcd.io/) system is used. The creators describe it as such:
> Etcd is a distributed, reliable key-value store for the most critical data of a distributed system.

In an effort to keep the focus on reliability and consistency, the authors of Etcd decided to allow only a single data type for both keys and values: byte arrays.
For more information on replication and persistence, see the Etcd documentation.
Mydis builds upon the solid Etcd framework to provide more data types and features such as atomic list operations and distributed locks.

Configuration
-------------
Without any configuration, Mydis starts up as a single node cluster on port 8383 with a storage limit of 2GB. This can be changed by creating a YAML configuration file named `mydis.conf` in either `/etc/mydis` or in the same directory as the executable.
To change the storage limit from the 2GB default, set `quota-backend-bytes` to the number of bytes the new limit should be. Clustering configuration is explained in another section.
It is recommended to leave `listen-client-urls` and `advertise-client-urls` as an empty list unless you want clients to be able to bypass Mydis and connect directly to Etcd.

See Etcd's documentation for more configuration options: https://coreos.com/etcd/docs/latest/op-guide/configuration.html

The listening port can be changed from the default os 8383 by specifying the environment variable `MYDIS_ADDRESS`. The default value is `0.0.0.0:8383`.

Clustering
----------
Clustering is handled entirely by Etcd. Its documentation explains the configuration required to create a cluster.
To summarize, clusters should always have an odd number of nodes.

Example cluster configuration:

**Nodes**
- `node1=10.0.0.1`
- `node2=10.0.0.2`
- `node3=10.0.0.3`

**Config for Node1**
```yaml
initial-advertise-peer-urls:
- https://10.0.0.1:2380

listen-peer-urls:
- https://10.0.0.1:2380

ClientAutoTLS: true
PeerAutoTLS: true

name: node1
initial-cluster: node1=https://10.0.0.1:2380,node2=https://10.0.0.2:2380,node3=https://10.0.0.3:2380
initial-cluster-token: name-of-cluster
```

**Config for Node2**
```yaml
initial-advertise-peer-urls:
- https://10.0.0.2:2380

listen-peer-urls:
- https://10.0.0.2:2380

ClientAutoTLS: true
PeerAutoTLS: true

name: node2
initial-cluster: node1=https://10.0.0.1:2380,node2=https://10.0.0.2:2380,node3=https://10.0.0.3:2380
initial-cluster-token: name-of-cluster
```

Some notes about this configuration:
- Nodes will communicate over an encrypted HTTPS socket using auto generated keys.
- Mydis will use the value of `ClientAutoTLS` or the client-specific TLS options to enable client encryption.
- It is **highly** recommended to always use `ClientAutoTLS: true` and `PeerAutoTLS: true` unless you specify your own TLS configuration. The client tries to use AutoTLS by default.
- The IP address `127.0.0.1` or `0.0.0.0` should **never** be used in any configuration option. The node's actual IP address should be used.
- The `name` is used in `initial-cluster` to determine which address in the list belongs to the current node, so it must match.
- The only fields that should change between nodes are: `initial-advertise-peer-urls`, `listen-peer-urls`, and `name`.
- While setting `initial-cluster-token` is optional, it's always a good idea to give each cluster a name.

See Etcd's clustering guide for more information: https://coreos.com/etcd/docs/latest/op-guide/clustering.html

Server API
----------
When used as a library, the server API uses protocol buffer messages with context, as required by gRPC.

Client API
----------
Some getter functions return a helper object, called `Value` that allows you to specify what data type you would like the response in. The data type functions available include:
- `Bytes()`
- `String()`
- `Bool()`
- `Proto(proto.Message)`
- `Int()`
- `Float()`
- `Time()`
- `Duration()`
- `List()`
- `Map()`

All of these functions return the desired data type and an error if there was a problem getting or converting the value.

Keys
----
Keys are strings of any length.

**Functions**
- `Keys() []string`: Get list of keys available in the cache.
- `KeysWithPrefix() []string`: Gets a list of keys with the given prefix.
- `Has(key) bool`: Determine if a key exists.
- `SetExpire(key, exp)`: Reset the expiration of a key to the number of seconds from now.
- `Delete(key)`: Delete a key.
- `Clear()`: Clear the cache.

Strings/Bytes
-------------
Strings can be text or byte arrays. Conversion between string and bytes is done without any additional memory allocations.

**Functions**
- `Get(key) Value`: Get a value, returns ErrKeyNotFound if key doesn't exist.
- `GetMany(keyList) map[string]Value`: Get multiple values.
- `GetWithPrefix(prefix) map[string]Value`: Gets the keys with the given prefix.
- `Set(key, value)`: Set a value.
- `SetNX(key, value) bool`: Set a value only if the key doesn't exist, returns true if changed.
- `SetMany(values) map[string]string`: Set many values, returning a map[key]errorText for any errors.
- `Length(key) int64`: Get the number of bytes stored at the given key.

Numbers
-------
Numbers can be 64-bit integers or floating-point values.

**Functions**
- `IncrementInt(key, by) int64`: Increment an integer and return new value, starts at zero if key doesn't exist.
- `IncrementFloat(key, by) float64`: Increment a float and return new value, starts at zero if key doesn't exist.
- `DecrementInt(key, by) int64`: Decrement an integer and return new value, starts at zero if key doesn't exist.
- `DecrementFloat(key, by) float64`: Decrement a float and return new value, starts at zero if key doesn't exist.

Lists
-----
Lists are lists of values.

**Functions**
- `GetListItem(key, index) Value`: Get a single item from a list by index, returns ErrKeyNotFound if key doesn't exist, or ErrorListIndexOutOfRange if index is out of range.
- `SetListItem(key, index, value)`: Set a single item in a list by index.
- `ListHas(key, value) int64`: Determines if a list has a value, returns index or -1 if not found.
- `ListLimit(key, limit)`: Sets the maximum length of a list, removing items from the top once limit is reached. Evaluated on insert and append only.
- `ListInsert(key, index, value)`: Insert an item in a list at the given index, creates new list and inserts item at index zero if key doesn't exist.
- `ListAppend(key, value)`: Append an item to the end of a list, creates new list if key doesn't exist.
- `ListPopLeft(key) Value`: Remove and return the first item in a list, returns ErrListEmpty if list is empty.
- `ListPopRight(key) Value`: Remove and return the last item in a list, returns ErrListEmpty if list is empty.
- `ListDelete(key, index)`: Remove an item from a list by index, returns an error if key or index doesn't exist.
- `ListDeleteItem(key, value) int64`: Search for and remove the first occurance of value from the list, returns index of item or -1 for not found.
- `ListLength(key) int64`: Get the number of items in a list.

Hashes
------
Hashes are objects with multiple string fields.

**Functions**
- `GetHashField(key, field) Value`: Get a single field from a hash, returns ErrHashFieldNotFound if field doesn't exist.
- `GetHashFields(key, fields) map[string]Value`: Get multiple fields from a hash, non-existent keys will not be added to the resulting map.
- `HashHas(key) bool`: Determines if a hash has a field, returns true or false.
- `HashFields(key) []string`: Get a list of hash fields.
- `HashValues(key) []Value`: Get a list of hash values.
- `HashLength(key) int64`: Get the number of fields in a hash.
- `SetHashField(key, field, value)`: Set a single field in a hash, creates new hash if key doesn't exist.
- `SetHashFields(key, values)`: Set multiple fields in a hash, creates new hash if key doesn't exist.
- `DelHashField(key, field)`: Delete a single field from a hash.

Locks
-----
Keys can be locked from modification.

**Functions**
- `Lock(key)`: Lock a key, waiting a default of 5 seconds if a lock already exists on the key before returning ErrKeyLocked.
- `LockWithTimeout(key, seconds)`: Lock a key, waiting for the given number of seconds if already locked before returning ErrKeyLocked.
- `Unlock(key)`: Unlock a key.
- `UnlockThenSet(key, value)`: Unlock a key, then immediately set its value.
- `SetLockTimeout(seconds)`: Sets the default timeout in seconds if key is already locked.

Events
------
Using the event handling feature, you can be notified when a key changes.

**Functions**
- `Watch(key, prefix)`: Get a notification event when a key changes. When calling one of the set functions, subscribed clients will be notified, including the sender if subscribed. If prefix is true, watches all keys with the given prefix.
- `UnWatch(key, prefix)`: Stop getting notifications when a key changes.
- `NewEventChannel()`: Returns a new Event channel.
- `CloseEventChannel(id)`: Closes an Event channel.

Usage
-----
**Server**
```go
server := mydis.NewServer(mydisNewServerConfig())
if err := server.Start(":8383"); err != nil {
	log.Fatalln(err)
}
```

**Client**
```go
client := mydis.NewClient(NewClientConfig("localhost:8383"))
s, err := client.Get("key").String()
if err != nil {
	return err
}
```

Expected in Next Release
------------------------
This is a list of changes expected to be implemented in the next release:
- Command-line utility
- Authentication
- Blocking list pop

Possible Future Enhancements
----------------------------
This is a list of possible enhancements that could be made in the future.
- SQL/Table-like functionality (think hashlist)

License
-------
Mydis is licensed under the Apache 2.0 license. See the [LICENSE] (LICENSE) file for details.