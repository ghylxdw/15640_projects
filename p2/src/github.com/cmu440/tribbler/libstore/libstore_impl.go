package libstore

/*
	cache 内存不够的问题
*/
import (
	"container/list"
	"errors"
	"fmt"
	"github.com/cmu440/tribbler/rpc/librpc"
	"github.com/cmu440/tribbler/rpc/storagerpc"
	"net/rpc"
	"strings"
	"sync"
	"time"
)

type leasetype struct {
	startFrom    time.Time
	validSeconds int
}

type valuelib struct {
	value string
	lease leasetype
}

type listlib struct {
	list  []string
	lease leasetype
}

type libstore struct {
	// client * rpc.Client
	Servers  []storagerpc.Node
	keyValue map[string]*valuelib
	keylist  map[string]*listlib

	mutex *sync.Mutex

	keyValudhistory map[string]*list.List
	keyListhistory  map[string]*list.List
	keyValueMutex   map[string]*sync.Mutex
	keyListMutex    map[string]*sync.Mutex
	mode            LeaseMode
	myHostPort      string
	clientmap       map[string]*rpc.Client
	// TODO: implement this!
}

// NewLibstore creates a new instance of a TribServer's libstore. masterServerHostPort
// is the master storage server's host:port. myHostPort is this Libstore's host:port
// (i.e. the callback address that the storage servers should use to send back
// notifications when leases are revoked).
//
// The mode argument is a debugging flag that determines how the Libstore should
// request/handle leases. If mode is Never, then the Libstore should never request
// leases from the storage server (i.e. the GetArgs.WantLease field should always
// be set to false). If mode is Always, then the Libstore should always request
// leases from the storage server (i.e. the GetArgs.WantLease field should always
// be set to true). If mode is Normal, then the Libstore should make its own
// decisions on whether or not a lease should be requested from the storage server,
// based on the requirements specified in the project PDF handout.  Note that the
// value of the mode flag may also determine whether or not the Libstore should
// register to receive RPCs from the storage servers.
//
// To register the Libstore to receive RPCs from the storage servers, the following
// line of code should suffice:
//
//     rpc.RegisterName("LeaseCallbacks", librpc.Wrap(libstore))
//
// Note that unlike in the NewTribServer and NewStorageServer functions, there is no
// need to create a brand new HTTP handler to serve the requests (the Libstore may
// simply reuse the TribServer's HTTP handler since the two run in the same process).
func NewLibstore(masterServerHostPort, myHostPort string, mode LeaseMode) (Libstore, error) {
	fmt.Println("New an libstore")

	client, err := rpc.DialHTTP("tcp", masterServerHostPort)
	if err != nil {
		// log.Fatalln("dialing:", err)
		return nil, errors.New("meet error when DialHTTP")
	}

	args := &storagerpc.GetServersArgs{}
	reply := &storagerpc.GetServersReply{}

	num := 0
	for reply.Status != storagerpc.OK {

		if num >= 5 {
			return nil, errors.New("cannot find the storage server")
		}

		err = client.Call("StorageServer.GetServers", args, reply)
		if err != nil {
			return nil, errors.New("meet error when client call")

		}
		if reply.Status != storagerpc.OK {
			time.Sleep(1000 * time.Millisecond)
		}
		num++
	}

	libstore := &libstore{
		Servers: reply.Servers,
		// keyValueMutex:     make(map[string]*sync.Mutex),
		// keyListMutex:      make(map[string]*sync.Mutex),
		mutex: &sync.Mutex{},

		// client: client,
		keyValue:        make(map[string]*valuelib),
		keylist:         make(map[string]*listlib),
		keyValudhistory: make(map[string]*list.List),
		keyListhistory:  make(map[string]*list.List),
		keyValueMutex:   make(map[string]*sync.Mutex),
		keyListMutex:    make(map[string]*sync.Mutex),
		mode:            mode,
		clientmap:       make(map[string]*rpc.Client),
		myHostPort:      myHostPort,
	}
	libstore.clientmap[masterServerHostPort] = client
	err = rpc.RegisterName("LeaseCallbacks", librpc.Wrap(libstore))
	// retry until register sucessfully
	for err != nil {
		err = rpc.RegisterName("LeaseCallbacks", librpc.Wrap(libstore))
	}

	go libstore.epochtime()
	return libstore, nil
	// return nil, errors.New("not implemented")

}

func (ls *libstore) initiateKeyValueMutex(key string) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()

	if ls.keyValueMutex[key] == nil {
		ls.keyValueMutex[key] = &sync.Mutex{}
	}
}

func (ls *libstore) initiateKeyListMutex(key string) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()

	if ls.keyListMutex[key] == nil {
		ls.keyListMutex[key] = &sync.Mutex{}
	}
}

func (ls *libstore) deleteinvalid() {
	now := time.Now()
	// ls.mutex.Lock()
	// // fmt.Println("deleteinvalid")
	// defer ls.mutex.Unlock()

	// deletelist := list.New()
	for key, value := range ls.keyValue {
		ls.initiateKeyValueMutex(key)
		ls.keyValueMutex[key].Lock()
		validUntil := value.lease.startFrom.Add(time.Duration(value.lease.validSeconds) * time.Second)
		if now.After(validUntil) {
			delete(ls.keyValue, key)
			// deletelist.PushBack(key)
		}
		ls.keyValueMutex[key].Unlock()
	}

	for key, value := range ls.keylist {
		ls.initiateKeyListMutex(key)
		ls.keyListMutex[key].Lock()
		validUntil := value.lease.startFrom.Add(time.Duration(value.lease.validSeconds) * time.Second)
		if now.After(validUntil) {
			delete(ls.keylist, key)
		}
		ls.keyListMutex[key].Unlock()
	}
}

func (ls *libstore) deletelist() {
	now := time.Now()
	// ls.mutex.Lock()
	// // fmt.Println("deleteinvalid")
	// defer ls.mutex.Unlock()

	for key, list := range ls.keyValudhistory {
		ls.initiateKeyValueMutex(key)
		ls.keyValueMutex[key].Lock()
		for list.Len() > 0 {
			e := list.Front()
			if e.Value.(time.Time).Add(time.Duration(storagerpc.QueryCacheSeconds)*time.Second).After(now) && list.Len() <= storagerpc.QueryCacheThresh {
				break
			}
			list.Remove(list.Front())
		}
		ls.keyValueMutex[key].Unlock()
	}

	for key, list := range ls.keyListhistory {
		ls.initiateKeyListMutex(key)
		ls.keyListMutex[key].Lock()
		for list.Len() > 0 {
			e := list.Front()
			if e.Value.(time.Time).Add(time.Duration(storagerpc.QueryCacheSeconds)*time.Second).After(now) && list.Len() <= storagerpc.QueryCacheThresh {
				break
			}
			list.Remove(list.Front())
		}
		ls.keyListMutex[key].Unlock()
	}
}

func (ls *libstore) epochtime() {
	for {
		select {
		case <-time.After(time.Duration(storagerpc.LeaseSeconds/2) * time.Second):
			ls.deleteinvalid()
		case <-time.After(time.Duration(storagerpc.QueryCacheSeconds/2) * time.Second):
			ls.deletelist()
		}

	}
}

func (ls *libstore) Findshashring(key string) string {
	serverkey := strings.Split(key, ":")
	hashring := StoreHash(serverkey[0])
	size := len(ls.Servers)
	var i int
	// fmt.Println("hashring")
	for i = 0; i < size; i++ {
		// fmt.Println(ls.Servers[i].NodeID)
		if ls.Servers[i].NodeID >= hashring {
			break
		}
	}
	if i == size {
		i = 0
	}
	// fmt.Println(i,size,"hashring")
	return ls.Servers[i].HostPort
}

func (ls *libstore) Get(key string) (string, error) {
	// fmt.Println("Get begin")
	now := time.Now()
	ls.initiateKeyValueMutex(key)
	ls.keyValueMutex[key].Lock()
	defer ls.keyValueMutex[key].Unlock()

	value, ok := ls.keyValue[key]

	if ok {

		validUntil := value.lease.startFrom.Add(time.Duration(value.lease.validSeconds) * time.Second)

		if now.After(validUntil) {
			delete(ls.keyValue, key)
			ok = !ok
		}
	}
	list, list_ok := ls.keyValudhistory[key]
	if list_ok {
		for list.Len() > 0 {
			e := list.Front()
			pretime := e.Value.(time.Time)
			// fmt.Println("pretime :"+ fmt.Sprintf("%d",pretime))
			judgetime := pretime.Add(time.Duration(storagerpc.QueryCacheSeconds) * time.Second)
			// fmt.Println("judgetime:"+ fmt.Sprintf("%d",judgetime))
			if judgetime.After(now) && list.Len() <= storagerpc.QueryCacheThresh {
				break
			}

			// fmt.Println("now:"+ fmt.Sprintf("%d",now))
			list.Remove(list.Front())

		}

		list.PushBack(now) // do not know wether count cache or not
	}
	if ok { //use cache
		return value.value, nil
	} else { // use the storage

		// fmt.Println("hotport:",server)
		want := false
		if ls.mode == Always {
			want = true
		}
		if ls.mode == Normal && list_ok && ls.keyValudhistory[key].Len() >= storagerpc.QueryCacheThresh {
			want = true
		}

		args := &storagerpc.GetArgs{
			Key:       key,
			WantLease: want,
			HostPort:  ls.myHostPort,
		}
		reply := &storagerpc.GetReply{}

		server := ls.Findshashring(key)
		client, ok := ls.clientmap[server]
		if !ok {

			clienttemp, err := rpc.DialHTTP("tcp", server)
			if err != nil {
				return "", errors.New("meet error when DialHTTP")
			}
			ls.clientmap[server] = clienttemp
			client = clienttemp
		}
		err := client.Call("StorageServer.Get", args, reply)
		if err != nil {

			return "", errors.New("error when libstore call get to storageserver")
		}

		if reply.Status == storagerpc.OK {
			if reply.Lease.Granted == true { // update cache
				ls.keyValue[key] = &valuelib{
					value: reply.Value,
					lease: leasetype{
						startFrom:    now,
						validSeconds: reply.Lease.ValidSeconds,
					},
				}
			}
		} else {
			return "", errors.New("reply not ready get")

		}
		return reply.Value, nil
	}
}

func (ls *libstore) Put(key, value string) error {

	ls.initiateKeyValueMutex(key)
	ls.keyValueMutex[key].Lock()
	_, ok := ls.keyValudhistory[key]
	if !ok {
		ls.keyValudhistory[key] = list.New()
	}
	ls.keyValueMutex[key].Unlock()

	args := &storagerpc.PutArgs{
		Key:   key,
		Value: value,
	}
	reply := &storagerpc.PutReply{}

	server := ls.Findshashring(key)
	client, ok := ls.clientmap[server]
	if !ok {

		clienttemp, err := rpc.DialHTTP("tcp", server)
		if err != nil {
			return errors.New("meet error when DialHTTP")
		}
		ls.clientmap[server] = clienttemp
		client = clienttemp
	}

	err := client.Call("StorageServer.Put", args, reply)
	if err != nil {
		return errors.New("error when libstore call get to StorageServer.Get")
	}
	if reply.Status == storagerpc.OK {

		return nil
	} else {
		return errors.New("put error libstore key not found")
	}

	return errors.New("not implemented")
}

func (ls *libstore) GetList(key string) ([]string, error) {
	now := time.Now()
	ls.initiateKeyListMutex(key)
	ls.keyListMutex[key].Lock()
	defer ls.keyListMutex[key].Unlock()

	value, ok := ls.keylist[key]

	if ok {
		validUntil := value.lease.startFrom.Add(time.Duration(value.lease.validSeconds) * time.Second)

		if now.After(validUntil) {
			delete(ls.keylist, key)
			ok = !ok
		}
	}
	list, list_ok := ls.keyListhistory[key]
	if list_ok {
		for list.Len() > 0 {
			e := list.Front()
			pretime := e.Value.(time.Time)
			judgetime := pretime.Add(time.Duration(storagerpc.QueryCacheSeconds) * time.Second)
			if judgetime.After(now) && list.Len() <= storagerpc.QueryCacheThresh {
				break
			}
			list.Remove(list.Front())
		}

		list.PushBack(now) // do not know wether count cache or not
	}
	if ok { //use cache
		return value.list, nil
	} else { // use the storage

		want := false
		if ls.mode == Always {
			want = true
		}
		if ls.mode == Normal && list_ok && ls.keyListhistory[key].Len() >= storagerpc.QueryCacheThresh {
			want = true
		}

		args := &storagerpc.GetArgs{
			Key:       key,
			WantLease: want,
			HostPort:  ls.myHostPort,
		}
		reply := &storagerpc.GetListReply{}

		server := ls.Findshashring(key)
		client, ok := ls.clientmap[server]
		if !ok {

			clienttemp, err := rpc.DialHTTP("tcp", server)
			if err != nil {
				return nil, errors.New("meet error when DialHTTP")
			}
			ls.clientmap[server] = clienttemp
			client = clienttemp
		}

		err := client.Call("StorageServer.GetList", args, &reply)
		if err != nil {

			return nil, errors.New("error when libstore call get to storageserver")
		}

		if reply.Status == storagerpc.OK {
			if reply.Lease.Granted == true { // update cache
				ls.keylist[key] = &listlib{
					list: reply.Value,
					lease: leasetype{
						startFrom:    now,
						validSeconds: reply.Lease.ValidSeconds,
					},
				}
			}
		} else if reply.Status == storagerpc.KeyNotFound {
			return make([]string, 0), nil
		} else {
			return nil, errors.New("reply not ready getlist: " + string(reply.Status))
		}
		return reply.Value, nil
	}
}

func (ls *libstore) AppendToList(key, newItem string) error {

	ls.initiateKeyListMutex(key)
	ls.keyListMutex[key].Lock()
	_, ok := ls.keyListhistory[key]
	if !ok {
		ls.keyListhistory[key] = list.New()
	}
	ls.keyListMutex[key].Unlock()

	args := &storagerpc.PutArgs{
		Key:   key,
		Value: newItem,
	}
	reply := &storagerpc.PutReply{}

	server := ls.Findshashring(key)
	client, ok := ls.clientmap[server]

	if !ok {

		clienttemp, err := rpc.DialHTTP("tcp", server)
		if err != nil {
			return errors.New("meet error when DialHTTP")
		}
		ls.clientmap[server] = clienttemp
		client = clienttemp
	}
	err := client.Call("StorageServer.AppendToList", args, reply)
	if err != nil {
		return err
	}
	if reply.Status == storagerpc.OK {
		return nil
	} else {
		return errors.New("AppendTolist error libstore key not found")
	}
	return errors.New("not implemented")
}

func (ls *libstore) RemoveFromList(key, removeItem string) error {

	args := &storagerpc.PutArgs{
		Key:   key,
		Value: removeItem,
	}
	reply := &storagerpc.PutReply{}

	server := ls.Findshashring(key)
	client, ok := ls.clientmap[server]
	if !ok {

		clienttemp, err := rpc.DialHTTP("tcp", server)
		if err != nil {
			return errors.New("meet error when DialHTTP")
		}
		ls.clientmap[server] = clienttemp
		client = clienttemp
	}

	err := client.Call("StorageServer.RemoveFromList", args, reply)
	if err != nil {
		return errors.New("error when libstore call get to StorageServer.Get")
	}
	if reply.Status == storagerpc.OK {
		return nil
	} else {
		return errors.New(" RemoveFromList error libstore key not found")
	}

	return errors.New("not implemented")
}

func (ls *libstore) RevokeLease(args *storagerpc.RevokeLeaseArgs, reply *storagerpc.RevokeLeaseReply) error {
	ls.initiateKeyValueMutex(args.Key)
	ls.initiateKeyListMutex(args.Key)
	ls.keyValueMutex[args.Key].Lock()
	defer ls.keyValueMutex[args.Key].Unlock()
	ls.keyListMutex[args.Key].Lock()
	defer ls.keyListMutex[args.Key].Unlock()

	_, okvalue := ls.keyValue[args.Key]
	_, oklist := ls.keylist[args.Key]
	if okvalue || oklist {
		reply.Status = storagerpc.OK
		if okvalue {
			delete(ls.keyValue, args.Key)
		} else {
			delete(ls.keylist, args.Key)
		}

	} else {
		reply.Status = storagerpc.KeyNotFound
	}

	return nil
}
