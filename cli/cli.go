package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/deejross/mydis"
)

var help = map[string][]string{
	"KEYS":            []string{"KEYS", "Get a list of keys in the cache"},
	"KEYSWITHPREFIX":  []string{"KEYSWITHPREFIX key", "Get a list of keys with the given prefix"},
	"HAS":             []string{"HAS key", "Checks if the cache has the given key"},
	"SETEXPIRE":       []string{"SETEXPIRE key duration", "Sets the expiration on a key"},
	"DELETE":          []string{"DELETE key", "Delete a key from the cache"},
	"CLEAR":           []string{"CLEAR", "Clear the cache"},
	"GET":             []string{"GET key", "Get a string from the cache"},
	"SET":             []string{"SET key value", "Set a string in the cache"},
	"SETNX":           []string{"SETNX key value", "Set a string in the cache only if the key doesn't already exist"},
	"SETINT":          []string{"SETINT key int", "Set an integeer in the cache"},
	"SETFLOAT":        []string{"SETFLOAT key float", "Set a float in the cache"},
	"INCREMENTINT":    []string{"INCREMENTINT key by", "Increment an integer by the given number and return the result"},
	"DECREMENTINT":    []string{"DECREMENTINT key by", "Decrement an integer by the given number and return the result"},
	"INCREMENTFLOAT":  []string{"INCREMENTFLOAT key by", "Increment a float by the given number and return the result"},
	"DECREMENTFLOAT":  []string{"DECREMENTFLOAT key by", "Decrement a float by the given number and return the result"},
	"GETLISTITEM":     []string{"GETLISTITEM key index", "Get a single item from a list by index"},
	"SETLISTITEM":     []string{"SETLISTITEM key index value", "Set a single item in a list by index"},
	"LISTLIMIT":       []string{"LISTLIMIT key limit", "Set the maximum length of a list, removing items from the top once reached"},
	"LISTLENGTH":      []string{"LISTLENGTH key", "Get the number of items in a list"},
	"LISTINSERT":      []string{"LISTINSERT key index value", "Insert a new item to a list at the given index"},
	"LISTAPPEND":      []string{"LISTAPPEND key value", "Insert a new item at the end of the list"},
	"LISTPOPLEFT":     []string{"LISTPOPLEFT key", "Returns and removes the first item in a list"},
	"LISTPOPRIGHT":    []string{"LISTPOPRIGHT key", "Returns and removes the last item in a list"},
	"LISTHAS":         []string{"LISTHAS key value", "Determines if a list contains an item"},
	"LISTDELETE":      []string{"LISTDELETE key index", "Removes an item from a list by index"},
	"LISTDELETEITEM":  []string{"LISTDELETEITEM key value", "Removes first occurance of value from a list, returns index of removed item or -1 for not found"},
	"GETHASHFIELD":    []string{"GETHASHFIELD key field", "Get a single value from a hash"},
	"HASHHAS":         []string{"HASHHAS key field", "Determines if a hash has a given field"},
	"HASHFIELDS":      []string{"HASHFIELDS key", "Get a list of the fields in a hash"},
	"HASHVALUES":      []string{"HASHVALUES key", "Get a list of the values in a hash"},
	"SETHASHFIELD":    []string{"SETHASHFIELD key field value", "Set a single value in a hash"},
	"DELHASHFIELD":    []string{"DELHASHFIELD key field", "Delete a field from a hash"},
	"LOCK":            []string{"LOCK key", "Lock a key"},
	"LOCKWITHTIMEOUT": []string{"LOCKWITHTIMEOUT key seconds", "Lock a key with custom timeout"},
	"UNLOCK":          []string{"UNLOCK key", "Unlock a key"},
	"SETLOCKTIMEOUT":  []string{"SETLOCKTIMEOUT seconds", "Set the default lock timeout"},
	"WATCH":           []string{"WATCH key", "Watch for changes to a key"},
	"UNWATCH":         []string{"UNWATCH key", "Unwatch for changes to a key"},
	"AUTHENABLE":      []string{"AUTHENABLE", "Enable authentication"},
	"AUTHDISABLE":     []string{"AUTHDISABLE", "Disable authentication"},
	"AUTHENTICATE":    []string{"AUTHENTICATE username password", "Authenticate a user"},
	"USERADD":         []string{"USERADD username password", "Add a new user"},
	"USERGET":         []string{"USERGET username", "Get detailed user information"},
	"USERLIST":        []string{"USERLIST", "Get list of all users"},
	"USERDELETE":      []string{"USERDELETE username", "Delete a specific user"},
	"USERCHANGEPASS":  []string{"USERCHANGEPASS username password", "Change the password for a user"},
	"USERGRANTROLE":   []string{"USERGRANTROLE username role", "Grant a role to a user"},
	"USERREVOKEROLE":  []string{"USERREVOKEROLE username role", "Revoke a role from a user"},
	"ROLEADD":         []string{"ROLEADD role", "Add a new role"},
	"ROLEGET":         []string{"ROLEGET role", "Get detailed information for a role"},
	"ROLELIST":        []string{"ROLELIST", "REturns a list of roles"},
	"ROLEDELETE":      []string{"ROLEDELETE role", "Delete a role"},
	"ROLEGRANTPERM":   []string{"ROLEGRANTPERM role key keyEnd"},
	"ROLEREVOKEPERM":  []string{"ROLEREVOKEPERM role key keyEnd"},
}

var closeCh = make(chan struct{})

func main() {
	argLen := len(os.Args)
	address := "localhost:8383"

	if argLen > 1 {
		address = os.Args[1]
		if !strings.Contains(address, ":") {
			address += ":8383"
		}
	}

	client, err := mydis.NewClient(mydis.NewClientConfig(address))
	if err != nil {
		writeErr(err)
		os.Exit(1)
	}

	if argLen > 2 {
		cmd := os.Args[2]
		args := []string{}
		for i := 3; i < argLen; i++ {
			args = append(args, os.Args[i])
		}
		if err := command(client, cmd, args); err != nil {
			writeErr(err)
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	} else {
		fmt.Println("Mydis Command Line Interface, Version:", mydis.VERSION)
		fmt.Println("Connected. Type 'help' for a list of commands.")
		startEventHandler(client)

		for prompt(client) {
		}

		os.Exit(0)
	}
}

func command(client *mydis.Client, cmd string, args []string) error {
	errNotEnoughArgs := errors.New("Not enough arguments")

	if cmd == "QUIT" {
		client.Close()
		return io.EOF
	} else if cmd == "HELP" {
		displayHelp(help)
		return nil
	} else if cmd == "KEYS" {
		result, err := client.Keys()
		if err != nil {
			return err
		}
		displayList(result)
		return err
	} else if cmd == "KEYSWITHPREFIX" {
		like := ""
		if len(args) >= 1 {
			like = args[0]
		}
		result, err := client.KeysWithPrefix(like)
		if err != nil {
			return err
		}
		displayList(result)
		return err
	} else if cmd == "HAS" {
		if len(args) >= 1 {
			result, err := client.Has(args[0])
			if err != nil {
				return err
			}
			fmt.Println(result)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "SETEXPIRE" {
		if len(args) >= 2 {
			d, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			return client.SetExpire(args[0], d)
		}
		return errNotEnoughArgs
	} else if cmd == "DELETE" {
		if len(args) >= 1 {
			return client.Delete(args[0])
		}
		return errNotEnoughArgs
	} else if cmd == "CLEAR" {
		return client.Clear()
	} else if cmd == "GET" {
		if len(args) >= 1 {
			result, err := client.Get(args[0]).String()
			if err != nil {
				return err
			}
			fmt.Println(result)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "GETMANY" {
		if len(args) >= 1 {
			result, err := client.GetMany(args)
			if err != nil {
				return err
			}
			displayMap(result)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "SET" {
		if len(args) >= 2 {
			return client.Set(args[0], args[1])
		}
		return errNotEnoughArgs
	} else if cmd == "SETNX" {
		if len(args) >= 2 {
			b, err := client.SetNX(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Println(b)
		}
		return errNotEnoughArgs
	} else if cmd == "SETINT" {
		if len(args) >= 2 {
			i, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			return client.Set(args[0], i)
		}
		return errNotEnoughArgs
	} else if cmd == "SETFLOAT" {
		if len(args) >= 2 {
			f, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				return err
			}
			return client.Set(args[0], f)
		}
		return errNotEnoughArgs
	} else if cmd == "INCREMENTINT" {
		if len(args) >= 2 {
			i, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			i, err = client.IncrementInt(args[0], i)
			if err != nil {
				return err
			}
			fmt.Println(i)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "DECREMENTINT" {
		if len(args) >= 2 {
			i, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			i, err = client.DecrementInt(args[0], i)
			if err != nil {
				return err
			}
			fmt.Println(i)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "INCREMENTFLOAT" {
		if len(args) >= 2 {
			f, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				return err
			}
			f, err = client.IncrementFloat(args[0], f)
			if err != nil {
				return err
			}
			fmt.Println(f)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "DECREMENTFLOAT" {
		if len(args) >= 2 {
			f, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				return err
			}
			f, err = client.DecrementFloat(args[0], f)
			if err != nil {
				return err
			}
			fmt.Println(f)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "GETLISTITEM" {
		if len(args) >= 2 {
			i, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			s, err := client.GetListItem(args[0], i).String()
			if err != nil {
				return err
			}
			fmt.Println(s)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "SETLISTITEM" {
		if len(args) >= 3 {
			i, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			return client.SetListItem(args[0], i, args[2])
		}
		return errNotEnoughArgs
	} else if cmd == "LISTLIMIT" {
		if len(args) >= 2 {
			i, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			return client.ListLimit(args[0], i)
		}
		return errNotEnoughArgs
	} else if cmd == "LISTLENGTH" {
		if len(args) >= 1 {
			i, err := client.ListLength(args[0])
			if err != nil {
				return err
			}
			fmt.Println(i)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "LISTINSERT" {
		if len(args) >= 3 {
			i, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			return client.ListInsert(args[0], i, args[2])
		}
		return errNotEnoughArgs
	} else if cmd == "LISTAPPEND" {
		if len(args) >= 2 {
			return client.ListAppend(args[0], args[1])
		}
		return errNotEnoughArgs
	} else if cmd == "LISTPOPLEFT" {
		if len(args) >= 1 {
			s, err := client.ListPopLeft(args[0]).String()
			if err != nil {
				return err
			}
			fmt.Println(s)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "LISTPOPRIGHT" {
		if len(args) >= 1 {
			s, err := client.ListPopRight(args[0]).String()
			if err != nil {
				return err
			}
			fmt.Println(s)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "LISTHAS" {
		if len(args) >= 2 {
			b, err := client.ListHas(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Println(b)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "LISTDELETE" {
		if len(args) >= 2 {
			i, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			return client.ListDelete(args[0], i)
		}
		return errNotEnoughArgs
	} else if cmd == "LISTDELETEITEM" {
		if len(args) >= 2 {
			i, err := client.ListDeleteItem(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Println(i)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "GETHASHFIELD" {
		if len(args) >= 2 {
			s, err := client.GetHashField(args[0], args[1]).String()
			if err != nil {
				return err
			}
			fmt.Println(s)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "HASHHAS" {
		if len(args) >= 2 {
			b, err := client.HashHas(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Println(b)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "HASHFIELDS" {
		if len(args) >= 1 {
			lst, err := client.HashFields(args[0])
			if err != nil {
				return err
			}
			displayList(lst)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "HASHVALUES" {
		if len(args) >= 1 {
			lst, err := client.HashValues(args[0])
			if err != nil {
				return err
			}
			displayValList(lst)
			return err
		}
		return errNotEnoughArgs
	} else if cmd == "SETHASHFIELD" {
		if len(args) >= 3 {
			return client.SetHashField(args[1], args[1], args[2])
		}
		return errNotEnoughArgs
	} else if cmd == "DELHASHFIELD" {
		if len(args) >= 2 {
			return client.DelHashField(args[0], args[1])
		}
		return errNotEnoughArgs
	} else if cmd == "SETLOCKTIMEOUT" {
		if len(args) >= 1 {
			d, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}
			client.SetLockTimeout(d)
			return nil
		}
		return errNotEnoughArgs
	} else if cmd == "WATCH" {
		if len(args) >= 1 {
			client.Watch(args[0], false)
			return nil
		}
		return errNotEnoughArgs
	} else if cmd == "UNWATCH" {
		if len(args) >= 1 {
			client.Unwatch(args[0], false)
			return nil
		}
		return errNotEnoughArgs
	} else if cmd == "AUTHENABLE" {
		return client.AuthEnable()
	} else if cmd == "AUTHDISABLE" {
		return client.AuthDisable()
	} else if cmd == "AUTHENTICATE" {
		if len(args) >= 2 {
			token, err := client.Authenticate(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Println(token)
		}
	}
	return errors.New("Unknown command: " + cmd)
}

func prompt(client *mydis.Client) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	s, err := reader.ReadString('\n')
	if err != nil {
		writeErr(err)
		return true
	}

	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return true
	}

	args := strings.Split(s, " ")
	cmd := strings.ToUpper(args[0])
	if len(args) > 1 {
		args = args[1:]
	} else {
		args = []string{}
	}

	if err := command(client, cmd, args); err == io.EOF {
		os.Exit(0)
	} else if err != nil {
		writeErr(err)
	}

	return true
}

func writeErr(err error) {
	io.WriteString(os.Stderr, err.Error()+"\n")
}

func displayHelp(result map[string][]string) {
	type cmd struct {
		usage string
		desc  string
	}

	commands := map[string]cmd{}
	names := []string{}
	for name, c := range result {
		names = append(names, name)
		commands[name] = cmd{usage: c[0], desc: c[1]}
	}
	sort.Strings(names)

	fmt.Println("Commands:")
	for _, name := range names {
		fmt.Println(name)
		fmt.Println("\tUsage:", commands[name].usage)
		fmt.Println("\tDesc: ", commands[name].desc)
	}

	fmt.Println("\nAdditional Commands:")
	fmt.Println("QUIT")
	fmt.Println("\tUsage: QUIT")
	fmt.Println("\tDesc: Quit program")
}

func displayList(result []string) {
	if len(result) == 0 {
		fmt.Println("")
	}

	for _, val := range result {
		fmt.Println(val)
	}
}

func displayValList(result []mydis.Value) {
	if len(result) == 0 {
		fmt.Println("")
	}

	for _, val := range result {
		s, _ := val.String()
		fmt.Println(s)
	}
}

func displayMap(result map[string]mydis.Value) {
	if len(result) == 0 {
		fmt.Println("")
	}

	keys := []string{}
	for key := range result {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	for _, key := range keys {
		fmt.Println(key+":", result[key])
	}
}

func startEventHandler(client *mydis.Client) {
	go func() {
		ch, _ := client.NewEventChannel()
		for {
			select {
			case <-closeCh:
				return
			case e := <-ch:
				t := mydis.Event_EventType_name[int32(e.Type)]
				if len(e.Current.Value) > 0 {
					fmt.Println("EVENT", t, e.Current.Key, mydis.BytesToString(e.Current.Value))
				} else {
					fmt.Println("EVENT", t, e.Current.Key)
				}
			}
		}
	}()
}
