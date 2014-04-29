package tribserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cmu440/tribbler/libstore"
	"github.com/cmu440/tribbler/rpc/tribrpc"
	"net"
	"net/http"
	"net/rpc"
	"sort"
	"sync"
	"time"
)

type tribServer struct {
	// TODO: implement this!
	mutex *sync.Mutex
	// user map[string] * User
	libstore libstore.Libstore
}

// // Len is part of sort.Interface.
// func (s *tribrpc.Tribble) Len() int {
// 	return len(s.planets)
// }

// // Swap is part of sort.Interface.
// func (s *tribrpc.Tribble) Swap(i, j int) {
// 	s.planets[i], s.planets[j] = s.planets[j], s.planets[i]
// }

// // Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
// func (s *tribrpc.Tribble) Less(i, j int) bool {
// 	return s.by(&s.planets[i], &s.planets[j])
// }

// type User struct{
// 	SubscriptUser map[string] bool
// 	tribble map[string] tribrpc.Tribble
// }

//user:tribid: tribble
//user:subuserid: userid

// NewTribServer creates, starts and returns a new TribServer. masterServerHostPort
// is the master storage server's host:port and port is this port number on which
// the TribServer should listen. A non-nil error should be returned if the TribServer
// could not be started.
//
// For hints on how to properly setup RPC, see the rpc/tribrpc package.
func NewTribServer(masterServerHostPort, myHostPort string) (TribServer, error) {
	// open listener
	fmt.Println("NewTribServer")
	libstore, err := libstore.NewLibstore(masterServerHostPort, myHostPort, libstore.Normal)
	if err != nil {
		return nil, errors.New("not implemented")
	}
	tribServer := &tribServer{
		libstore: libstore,
	}

	listener, err := net.Listen("tcp", myHostPort)
	for err != nil {
		listener, err = net.Listen("tcp", myHostPort)
	}
	// register RPC
	err = rpc.RegisterName("TribServer", tribrpc.Wrap(tribServer))
	// repeat in case not register succesfully
	for err != nil {
		err = rpc.RegisterName("TribServer", tribrpc.Wrap(tribServer))
	}
	if err != nil {
		return nil, err
	}
	// bind listener with the RPC service
	rpc.HandleHTTP()
	go http.Serve(listener, nil)

	return tribServer, nil
}

func (ls *tribServer) initiateKeyValueMutex() {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
}

func (ts *tribServer) CreateUser(args *tribrpc.CreateUserArgs, reply *tribrpc.CreateUserReply) error {
	// ts.initiateKeyValueMutex()
	key := args.UserID + ":user"
	_, err := ts.libstore.Get(key)
	if err != nil {
		reply.Status = tribrpc.OK
		ts.libstore.Put(key, key)
	} else {
		reply.Status = tribrpc.Exists
	}

	return nil
}

func (ts *tribServer) AddSubscription(args *tribrpc.SubscriptionArgs, reply *tribrpc.SubscriptionReply) error {
	_, ok_user := ts.libstore.Get(args.UserID + ":user")
	_, ok_target := ts.libstore.Get(args.TargetUserID + ":user")
	if ok_user != nil {
		reply.Status = tribrpc.NoSuchUser
	} else if ok_target != nil {
		reply.Status = tribrpc.NoSuchTargetUser
	} else {
		key := args.UserID + ":subscribe_list"
		value := args.TargetUserID

		err := ts.libstore.AppendToList(key, value)
		if err != nil {
			reply.Status = tribrpc.Exists
		} else {
			reply.Status = tribrpc.OK
		}
	}
	return nil
}

func (ts *tribServer) RemoveSubscription(args *tribrpc.SubscriptionArgs, reply *tribrpc.SubscriptionReply) error {
	_, ok_user := ts.libstore.Get(args.UserID + ":user")
	_, ok_target := ts.libstore.Get(args.TargetUserID + ":user")
	if ok_user != nil {
		reply.Status = tribrpc.NoSuchUser
	} else if ok_target != nil {
		reply.Status = tribrpc.NoSuchTargetUser
	} else {
		key := args.UserID + ":subscribe_list"
		value := args.TargetUserID

		err := ts.libstore.RemoveFromList(key, value)
		if err != nil {
			reply.Status = tribrpc.NoSuchTargetUser
		} else {
			reply.Status = tribrpc.OK
		}
	}

	return nil
}

func (ts *tribServer) GetSubscriptions(args *tribrpc.GetSubscriptionsArgs, reply *tribrpc.GetSubscriptionsReply) error {
	_, ok_user := ts.libstore.Get(args.UserID + ":user")
	if ok_user != nil {
		reply.Status = tribrpc.NoSuchUser
	} else {
		key := args.UserID + ":subscribe_list"

		value, err := ts.libstore.GetList(key)
		if err != nil {
			return err
		} else {
			reply.Status = tribrpc.OK
			reply.UserIDs = value
		}

	}
	return nil
}

func (ts *tribServer) PostTribble(args *tribrpc.PostTribbleArgs, reply *tribrpc.PostTribbleReply) error {

	_, ok_user := ts.libstore.Get(args.UserID + ":user")
	if ok_user != nil {
		reply.Status = tribrpc.NoSuchUser
	} else {
		reply.Status = tribrpc.OK
		tribble := &tribrpc.Tribble{
			UserID:   args.UserID,
			Posted:   time.Now(),
			Contents: args.Contents,
		}
		tribbleid := time.Now().UTC().UnixNano()
		temp, _ := json.Marshal(tribble)

		err := ts.libstore.Put(args.UserID+":"+fmt.Sprintf("%d", tribbleid), string(temp[:]))
		if err != nil {
			return errors.New("error PostTribble")
		}

		err = ts.libstore.AppendToList(args.UserID+":tribble_list", fmt.Sprintf("%d", tribbleid))
		if err != nil {
			return errors.New("error AppendToList " + err.Error())
		}
	}
	return nil
}

type tList []tribrpc.Tribble

func (l tList) Len() int           { return len(l) }
func (l tList) Less(i, j int) bool { return l[i].Posted.After(l[j].Posted) }
func (l tList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

func (ts *tribServer) GetTribblesOneuser(UserID string) ([]tribrpc.Tribble, error) {
	tribble_list, err := ts.libstore.GetList(UserID + ":tribble_list")
	if len(tribble_list) != 0 && err != nil {
		return nil, errors.New("error GetTribbles")
	}
	var tribble tribrpc.Tribble

	tribbles := make([]tribrpc.Tribble, len(tribble_list))
	valueChan := make(chan string, 10)

	getTribbleFromId := func(libStore libstore.Libstore, tribbleId string, ch chan string) {
		val, _ := ts.libstore.Get(tribbleId)
		ch <- val
	}
	for i := 0; i < len(tribble_list); i++ {
		go getTribbleFromId(ts.libstore, UserID+":"+tribble_list[i], valueChan)
	}
	for i := 0; i < len(tribble_list); i++ {
		value := <-valueChan
		json.Unmarshal([]byte(value), &tribble)
		tribbles[i] = tribble
	}

	return tribbles, nil

}

func (ts *tribServer) GetTribbles(args *tribrpc.GetTribblesArgs, reply *tribrpc.GetTribblesReply) error {
	_, ok_user := ts.libstore.Get(args.UserID + ":user")
	if ok_user != nil {
		reply.Status = tribrpc.NoSuchUser
	} else {
		reply.Status = tribrpc.OK
		tribbles, err := ts.GetTribblesOneuser(args.UserID)
		if err != nil {
			return err
		}
		sort.Sort(tList(tribbles))
		length := len(tribbles)
		if length > 100 {
			length = 100
		}
		reply.Tribbles = make([]tribrpc.Tribble, length)
		for i := 0; i < length; i++ {
			reply.Tribbles[i] = tribbles[i]
		}
	}
	return nil
}

func (ts *tribServer) GetTribblesBySubscription(args *tribrpc.GetTribblesArgs, reply *tribrpc.GetTribblesReply) error {
	_, ok_user := ts.libstore.Get(args.UserID + ":user")
	if ok_user != nil {
		reply.Status = tribrpc.NoSuchUser
	} else {
		reply.Status = tribrpc.OK
		key := args.UserID + ":subscribe_list"

		subscribe_list, err := ts.libstore.GetList(key)
		if len(subscribe_list) != 0 && err != nil {
			return errors.New("error GetTribblesBySubscription")
		}

		valuechan := make(chan []tribrpc.Tribble, 10)

		ans := make([]tribrpc.Tribble, 0)
		go func() {
			for i := 0; i < len(subscribe_list); i++ {
				userid := i
				go func() {
					tribbles, _ := ts.GetTribblesOneuser(subscribe_list[userid])

					valuechan <- tribbles
				}()
			}
		}()

		for i := 0; i < len(subscribe_list); i++ {
			value := <-valuechan
			ans = append(ans, value...)
		}

		sort.Sort(tList(ans))
		length := len(ans)
		if length > 100 {
			length = 100
		}
		reply.Tribbles = make([]tribrpc.Tribble, length)
		for i := 0; i < length; i++ {
			reply.Tribbles[i] = ans[i]
		}
	}
	return nil
}
