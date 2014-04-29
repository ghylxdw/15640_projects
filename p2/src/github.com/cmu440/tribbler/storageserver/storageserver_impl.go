package storageserver

import (
	"fmt"

	//"fmt"

	"container/list"
	"errors"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cmu440/tribbler/libstore"
	"github.com/cmu440/tribbler/rpc/storagerpc"
)

const (
	localHost = "127.0.0.1"
)

type lease struct {
	startFrom    time.Time
	validSeconds int
}

type valueStoreType struct {
	value        string
	leaseGranted map[string]*lease
}

type listStoreType struct {
	list         *list.List
	leaseGranted map[string]*lease
}

type storageServer struct {
	status            storagerpc.Status
	keyValueStore     map[string]*valueStoreType
	keyListStore      map[string]*listStoreType
	isMaster          bool
	nodeID            uint32
	numNodes          int
	serversList       []storagerpc.Node
	serversMap        map[string]storagerpc.Node
	mutex             *sync.Mutex
	keyValueMutex     map[string]*sync.Mutex
	keyListMutex      map[string]*sync.Mutex
	clientCache       map[string]*rpc.Client
	clientCacheMutex  map[string]*sync.Mutex
	masterReadySignal chan struct{}
}

// NewStorageServer creates and starts a new StorageServer. masterServerHostPort
// is the master storage server's host:port address. If empty, then this server
// is the master; otherwise, this server is a slave. numNodes is the total number of
// servers in the ring. port is the port number that this server should listen on.
// nodeID is a random, unsigned 32-bit ID identifying this server.
//
// This function should return only once all storage servers have joined the ring,
// and should return a non-nil error if the storage server could not be started.
func NewStorageServer(masterServerHostPort string, numNodes, port int, nodeID uint32) (StorageServer, error) {
	storageServer := &storageServer{
		status:            storagerpc.NotReady,
		keyValueStore:     make(map[string]*valueStoreType),
		keyListStore:      make(map[string]*listStoreType),
		isMaster:          false,
		nodeID:            nodeID,
		numNodes:          numNodes,
		serversList:       make([]storagerpc.Node, numNodes),
		serversMap:        make(map[string]storagerpc.Node),
		mutex:             &sync.Mutex{},
		keyValueMutex:     make(map[string]*sync.Mutex),
		keyListMutex:      make(map[string]*sync.Mutex),
		clientCache:       make(map[string]*rpc.Client),
		clientCacheMutex:  make(map[string]*sync.Mutex),
		masterReadySignal: make(chan struct{}),
	}
	if masterServerHostPort == "" {
		storageServer.isMaster = true
	}

	//fmt.Println("server creating...")
	if numNodes <= 0 {
		return nil, errors.New("numNodes illegal")
	}

	// open listener until success
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	for err != nil {
		listener, err = net.Listen("tcp", ":"+strconv.Itoa(port))
	}

	// register RPC until success
	err = rpc.RegisterName("StorageServer", storagerpc.Wrap(storageServer))
	for err != nil {
		err = rpc.RegisterName("StorageServer", storagerpc.Wrap(storageServer))
	}

	// bind listener with the RPC service
	rpc.HandleHTTP()
	go http.Serve(listener, nil)

	// create args and reply for the server itself
	args := &storagerpc.RegisterArgs{ServerInfo: storagerpc.Node{HostPort: localHost + ":" + strconv.Itoa(port), NodeID: nodeID}}
	reply := &storagerpc.RegisterReply{}

	// if it's the master
	if storageServer.isMaster {

		// register the master itself
		go storageServer.RegisterServer(args, reply)
		// wait for the signal from RegisterServer method when the master server is ready
		<-storageServer.masterReadySignal
	} else {
		//fmt.Println("slave server creating...")
		client, err := rpc.DialHTTP("tcp", masterServerHostPort)
		if err != nil {
			return nil, errors.New("cannot create connection to the master server")
		}
		for err != nil || reply.Status != storagerpc.OK {
			err = client.Call("StorageServer.RegisterServer", args, reply)

			if reply.Status != storagerpc.OK {
				time.Sleep(1000 * time.Millisecond)
			}
		}
		// no need to lock here if RegisterServer method runs only on master server
		storageServer.mutex.Lock()
		storageServer.serversList = reply.Servers
		storageServer.numNodes = len(reply.Servers)
		storageServer.status = storagerpc.OK
		storageServer.mutex.Unlock()
	}

	//fmt.Println("storage server ready!")
	return storageServer, nil
}

func (ss *storageServer) RegisterServer(args *storagerpc.RegisterArgs, reply *storagerpc.RegisterReply) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	//fmt.Println("Register Server Started...")
	// check if this method is executed on the master server
	if !ss.isMaster {
		return errors.New("cannot register server on a slave server")
	}

	slaveServerInfo := args.ServerInfo
	_, exist := ss.serversMap[slaveServerInfo.HostPort]
	ss.serversMap[slaveServerInfo.HostPort] = slaveServerInfo

	if len(ss.serversMap) > ss.numNodes {
		return errors.New("num of servers registered exceeds the num of nodes specified")
	}

	if len(ss.serversMap) == ss.numNodes {
		//fmt.Println("numNodes match")
		if !exist {
			//fmt.Println("not exist")
			index := 0
			for _, v := range ss.serversMap {
				ss.serversList[index] = v
				index++
			}

			// use selective sort (the numNodes should be relatively small) to sort the server list to ascending order on their nodeId
			for i := 0; i < ss.numNodes; i++ {
				for j := i + 1; j < ss.numNodes; j++ {
					if ss.serversList[i].NodeID > ss.serversList[j].NodeID {
						tmp := ss.serversList[i]
						ss.serversList[i] = ss.serversList[j]
						ss.serversList[j] = tmp
					}
				}
			}
			// notify the NewStorageServer method
			ss.status = storagerpc.OK
			ss.masterReadySignal <- struct{}{}
			//fmt.Println("signal sent")
		}
		reply.Servers = ss.serversList
		reply.Status = storagerpc.OK
	}

	if len(ss.serversMap) < ss.numNodes {
		reply.Servers = nil
		reply.Status = storagerpc.NotReady
	}

	//fmt.Println("Register server complete!")
	return nil
}

func (ss *storageServer) GetServers(args *storagerpc.GetServersArgs, reply *storagerpc.GetServersReply) error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if !ss.isMaster {
		return errors.New("cannot get servers on a slave server")
	}

	if ss.status == storagerpc.OK {
		reply.Servers = ss.serversList
	}

	reply.Status = ss.status
	//fmt.Println("GetServers: ", reply.Status)

	return nil
}

func (ss *storageServer) Get(args *storagerpc.GetArgs, reply *storagerpc.GetReply) error {
	// check if the storage server is ready
	isReady := ss.checkServerIsReady()
	if !isReady {
		reply.Status = storagerpc.NotReady
		return nil
	}

	// check if the Get request go to the right server
	isRightServer := ss.checkIsRightServer(args.Key)
	if !isRightServer {
		reply.Status = storagerpc.WrongServer
		return nil
	}

	// initiate the mutex for specified key if necessary
	ss.initiateKeyValueMutex(args.Key)
	ss.keyValueMutex[args.Key].Lock()
	defer ss.keyValueMutex[args.Key].Unlock()

	//fmt.Println("GET: ", args.Key, args.WantLease)
	valueStore, exist := ss.keyValueStore[args.Key]
	if !exist {
		reply.Status = storagerpc.KeyNotFound
	} else {
		reply.Value = valueStore.value
		reply.Status = storagerpc.OK
		if args.WantLease {
			// update lease in the storage server and grant lease to the client if necessary
			reply.Lease.Granted, reply.Lease.ValidSeconds = ss.grantLeaseIfNecessary(args.HostPort, valueStore.leaseGranted)
		}
	}

	return nil
}

// the logic of this function is nearly the same as the Get() function
func (ss *storageServer) GetList(args *storagerpc.GetArgs, reply *storagerpc.GetListReply) error {
	// check if the storage server is ready
	isReady := ss.checkServerIsReady()
	if !isReady {
		reply.Status = storagerpc.NotReady
		return nil
	}

	// check if the Get request go to the right server
	isRightServer := ss.checkIsRightServer(args.Key)
	if !isRightServer {
		reply.Status = storagerpc.WrongServer
		return nil
	}

	ss.initiateKeyListMutex(args.Key)

	ss.keyListMutex[args.Key].Lock()
	defer ss.keyListMutex[args.Key].Unlock()

	listStore, exist := ss.keyListStore[args.Key]
	if !exist {
		reply.Status = storagerpc.KeyNotFound
	} else {
		// convert list to slice
		list := listStore.list
		retSlice := make([]string, list.Len())
		index := 0
		for e := list.Front(); e != nil; e = e.Next() {
			retSlice[index] = e.Value.(string)
			index++
		}

		reply.Value = retSlice
		reply.Status = storagerpc.OK
		if args.WantLease {
			reply.Lease.Granted, reply.Lease.ValidSeconds = ss.grantLeaseIfNecessary(args.HostPort, listStore.leaseGranted)
		}
	}

	return nil
}

func (ss *storageServer) Put(args *storagerpc.PutArgs, reply *storagerpc.PutReply) error {
	// check if the storage server is ready
	isReady := ss.checkServerIsReady()
	if !isReady {
		reply.Status = storagerpc.NotReady
		return nil
	}
	// check if the Get request go to the right server
	isRightServer := ss.checkIsRightServer(args.Key)
	if !isRightServer {
		reply.Status = storagerpc.WrongServer
		return nil
	}

	ss.initiateKeyValueMutex(args.Key)
	ss.keyValueMutex[args.Key].Lock()
	defer ss.keyValueMutex[args.Key].Unlock()

	//fmt.Println("PUT: ", args.Key, args.Value)
	valueStore, exist := ss.keyValueStore[args.Key]
	if !exist {
		ss.keyValueStore[args.Key] = &valueStoreType{value: args.Value, leaseGranted: make(map[string]*lease)}
	} else {
		// revoke all previous leases before updating the value
		err := ss.revokeGrantedLease(args.Key, valueStore.leaseGranted)
		if err != nil {
			return err
		}
		// if all granted leases are revoked successfully,
		valueStore.leaseGranted = make(map[string]*lease)
		// update the value that corresponds to the key
		valueStore.value = args.Value
	}
	reply.Status = storagerpc.OK
	//fmt.Println("PUT REPLY: ", reply.Status)
	return nil
}

func (ss *storageServer) AppendToList(args *storagerpc.PutArgs, reply *storagerpc.PutReply) error {
	// check if the storage server is ready
	isReady := ss.checkServerIsReady()
	if !isReady {
		reply.Status = storagerpc.NotReady
		return nil
	}
	// check if the Get request go to the right server
	isRightServer := ss.checkIsRightServer(args.Key)
	if !isRightServer {
		reply.Status = storagerpc.WrongServer
		return nil
	}

	ss.initiateKeyListMutex(args.Key)
	ss.keyListMutex[args.Key].Lock()
	defer ss.keyListMutex[args.Key].Unlock()

	listStore, exist := ss.keyListStore[args.Key]
	if !exist {
		// create a new listStore and put the value into the list
		newListStore := &listStoreType{list: list.New(), leaseGranted: make(map[string]*lease)}
		newListStore.list.PushBack(args.Value)
		ss.keyListStore[args.Key] = newListStore

		reply.Status = storagerpc.OK
	} else {
		// revoke all previous leases before updating the value
		err := ss.revokeGrantedLease(args.Key, listStore.leaseGranted)
		if err != nil {
			return err
		}

		list := listStore.list
		itemExist := false
		for e := list.Front(); e != nil; e = e.Next() {
			if e.Value.(string) == args.Value {
				itemExist = true
				break
			}
		}
		if itemExist {
			reply.Status = storagerpc.ItemExists
		} else {
			list.PushBack(args.Value)
			reply.Status = storagerpc.OK
		}
		listStore.leaseGranted = make(map[string]*lease)
	}
	return nil
}

func (ss *storageServer) RemoveFromList(args *storagerpc.PutArgs, reply *storagerpc.PutReply) error {
	// check if the storage server is ready
	isReady := ss.checkServerIsReady()
	if !isReady {
		reply.Status = storagerpc.NotReady
		return nil
	}
	// check if the Get request go to the right server
	isRightServer := ss.checkIsRightServer(args.Key)
	if !isRightServer {
		reply.Status = storagerpc.WrongServer
		return nil
	}

	ss.initiateKeyListMutex(args.Key)
	ss.keyListMutex[args.Key].Lock()
	defer ss.keyListMutex[args.Key].Unlock()

	listStore, exist := ss.keyListStore[args.Key]
	if !exist {
		reply.Status = storagerpc.ItemNotFound
	} else {
		// revoke all previous leases before updating the value
		err := ss.revokeGrantedLease(args.Key, listStore.leaseGranted)
		if err != nil {
			return err
		}

		list := listStore.list
		itemFound := false
		for e := list.Front(); e != nil; e = e.Next() {
			if e.Value.(string) == args.Value {
				itemFound = true
				list.Remove(e)
				break
			}
		}
		if itemFound {
			reply.Status = storagerpc.OK
		} else {
			reply.Status = storagerpc.ItemNotFound
		}
		listStore.leaseGranted = make(map[string]*lease)
	}
	return nil
}

func (ss *storageServer) checkServerIsReady() bool {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	return ss.status == storagerpc.OK
}

func (ss *storageServer) initiateKeyValueMutex(key string) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if ss.keyValueMutex[key] == nil {
		ss.keyValueMutex[key] = &sync.Mutex{}
	}
}

func (ss *storageServer) initiateKeyListMutex(key string) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if ss.keyListMutex[key] == nil {
		ss.keyListMutex[key] = &sync.Mutex{}
	}
}

func (ss *storageServer) initiateClientCacheMutex(key string) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if ss.clientCacheMutex[key] == nil {
		ss.clientCacheMutex[key] = &sync.Mutex{}
	}
}

func (ss *storageServer) checkIsRightServer(key string) bool {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	partitionKey := strings.Split(key, ":")[0]
	hashVal := libstore.StoreHash(partitionKey)
	nodeIDBelong := ss.serversList[0].NodeID
	for _, node := range ss.serversList {
		if hashVal <= node.NodeID {
			nodeIDBelong = node.NodeID
			break
		}
	}

	return nodeIDBelong == ss.nodeID
}

func (ss *storageServer) grantLeaseIfNecessary(hostPort string, leaseGranted map[string]*lease) (granted bool, validSeconds int) {
	prevLease, prevLeaseExist := leaseGranted[hostPort]
	now := time.Now()
	// if there is a granted lease for the hostport
	if prevLeaseExist {
		validUntil := prevLease.startFrom.Add(time.Duration(prevLease.validSeconds) * time.Second)
		// if the previous granted lease is expired, update the expired lease and grant new lease to the client
		if now.After(validUntil) {
			prevLease.startFrom = now
			prevLease.validSeconds = storagerpc.LeaseSeconds + storagerpc.LeaseGuardSeconds
			granted = true
			validSeconds = storagerpc.LeaseSeconds
		} else { // else refuse to grant lease
			granted = false
			validSeconds = 0
		}
	} else {
		// store new lease on server
		leaseGranted[hostPort] = &lease{now, storagerpc.LeaseSeconds + storagerpc.LeaseGuardSeconds}
		// grant new lease to the client
		granted = true
		validSeconds = storagerpc.LeaseSeconds
	}
	return
}

func (ss *storageServer) revokeGrantedLease(revokedKey string, leaseGranted map[string]*lease) error {
	//fmt.Println("Starting revoking leases...")
	for hostport, revokedLease := range leaseGranted {
		validUntil := revokedLease.startFrom.Add(time.Duration(revokedLease.validSeconds) * time.Second)
		// revoke the lease if the lease is not expired
		now := time.Now()
		if now.Before(validUntil) {
			var err error
			ss.initiateClientCacheMutex(hostport)
			ss.clientCacheMutex[hostport].Lock()
			client, ok := ss.clientCache[hostport]
			if !ok {
				client, err = rpc.DialHTTP("tcp", hostport)
				if err != nil {
					fmt.Println("error dialing")
					return err
				}
				ss.clientCache[hostport] = client
			}
			ss.clientCacheMutex[hostport].Unlock()

			expireTimer := func(c chan struct{}, timeToExpire time.Duration) {
				time.Sleep(timeToExpire)
				c <- struct{}{}
			}

			revokeRpcCall := func(c chan struct{}, args *storagerpc.RevokeLeaseArgs, reply *storagerpc.RevokeLeaseReply, cli *rpc.Client) {
				err := cli.Call("LeaseCallbacks.RevokeLease", args, reply)
				if err == nil && (reply.Status == storagerpc.OK || reply.Status == storagerpc.KeyNotFound) {
					c <- struct{}{}
				}
			}

			args := &storagerpc.RevokeLeaseArgs{Key: revokedKey}
			reply := &storagerpc.RevokeLeaseReply{}
			c := make(chan struct{}, 1)
			go expireTimer(c, validUntil.Sub(now))
			go revokeRpcCall(c, args, reply, client)
			<-c
		}
	}
	return nil
}
