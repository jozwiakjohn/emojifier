// john jozwiak 2018 Jul 30 for Modulus on-site interview warmup.

package main

import (
  "fmt"
  "os"
  "net"
  "bufio"
  "sort"
  "strings"
  "time"
  "syscall"
  "regexp"
  "strconv"
//"./bbolt"
//  https://github.com/aws/aws-sdk-go  //  time permitting, i might play with this.
)

var (
  serverIsActive bool      = false
  mmappedFile []byte       = nil
  mmappedFileLength  int64 = 0
  offsetOfLine             = make( map[int]int )
  spinnerstate       uint8 = 0
  spinnerRunning     bool  = false
  verbosity          bool  = false
  matchesQuitRegex         = regexp.MustCompile( `QUIT`)
  matchesShutdownRegex     = regexp.MustCompile( `SHUTDOWN`)
  matchesGetNumberRegex    = regexp.MustCompile( `GET [0-9]+`)
  matchesNumberRegex       = regexp.MustCompile( `^[0-9]+$`)
  matchesOKRegex           = regexp.MustCompile( `OK`)
  cheeseburger             = regexp.MustCompile( "[Cc]heeseburger")
  hotdog                   = regexp.MustCompile( "[Hh]otdog")
  dogOrPuppy               = regexp.MustCompile( "([Dd]og)|([Pp]uppy)")
  catOrKitten              = regexp.MustCompile( "([Cc]at)|([Kk]itten)")
)

const versionString string = "%v's version is 2018.07.30\n"

/////////////////////////////////////////////////

//  fun with tail recursive traversal of commandline arguments.
//  for instance,
//    scottsdale help version help server whatever.txt -p 8910
//  should work.

func  directions( args []string ) {

  var cmdname string = os.Args[0]
  var message string = `
   %s is John Jozwiak's effort to warm up for Modulus onsite interviewing.

      %s help                         ... prints this message.
      %s version                      ... prints the version of %s.
      %s server <filename> -p <port>  ... runs in line server mode per the assignment listening on port N.
      %s client <host:port>           ... connect as a client to a server at host and port, entering a REPL.

   Copyright 2018 John Jozwiak (send requests to jozwiakjohn@gmail.com) 

`
  fmt.Printf( message , cmdname , cmdname , cmdname , cmdname , cmdname , cmdname )

  if len(args) > 0 { processCommandLine( args ) }
}

/////////////////////////////////////////////////

func  version( args []string) {

  fmt.Printf( versionString,os.Args[0] )
  fmt.Printf( "len(\"OK\r\n\") = %d\n", len("OK\r\n") )
  if len(args) > 0 { processCommandLine( args ) }
}

/////////////////////////////////////////////////

func emojify( line string ) string {

  switch {
    case  strings.Contains(line,"cheeseburger") :
      return emojify( cheeseburger.ReplaceAllString(line,"🍔") )

    case  strings.Contains(line,"hotdog") :
      return emojify( hotdog.ReplaceAllString(line,"🌭") )

    case  strings.Contains(line,"dog") , strings.Contains(line,"puppy") :
      return emojify( dogOrPuppy.ReplaceAllString(line,"🐕") )

    case  strings.Contains(line,"cat") , strings.Contains(line,"kitten") :
      return emojify( catOrKitten.ReplaceAllString(line,"🐯") )

    default:
      return line
  }
}

/////////////////////////////////////////////////

var spin_spinner = func() {

  spinnerRunning = true

  var k string // rune is a pretty goofy term for the idea of a UTF-8 character.

  for spinnerRunning {

    switch spinnerstate {
      case 0 : k = "\\"
      case 1 : k = "|"
      case 2 : k = "/"
      case 3 : k = "-"
    }
    spinnerstate = (spinnerstate + 1) % 4

    fmt.Printf("\r%v",k)
    time.Sleep( 330 * time.Millisecond )
  }

  fmt.Println()
}

/////////////////////////////////////////////////

func  interactWithClient( connection net.Conn ) {

  var connectionReader = bufio.NewReader(connection)
  var active bool = true
  for active {
    requestString,_ := connectionReader.ReadString('\n')
    requestString   =  strings.TrimRight( requestString,"\r\n")
    fmt.Printf("responding to \"%v\"...\n",requestString)

    matchesQuit     := matchesQuitRegex.MatchString( requestString)
    matchesShutdown := matchesShutdownRegex.MatchString( requestString)
    matchesNumber   := matchesGetNumberRegex.MatchString( requestString)

    switch {
      case matchesQuit:
        active = false
        break //  nothing to do here?

      case matchesShutdown:
        serverIsActive = false
        os.Exit(0)  //  kind of harsh.

      case matchesNumber:
        nnnn,err := strconv.Atoi(requestString[4:])
        if err == nil {
          nnnn--
          if (nnnn >= 0) && (nnnn < len(offsetOfLine) - 1) {

            var startOffset int = offsetOfLine[int(nnnn  )]
            var stopOffset  int = offsetOfLine[int(nnnn+1)]-1
            var original []byte = mmappedFile[startOffset:stopOffset]
            const okeydoke      = "OK\r\n"
            fmt.Fprintf(connection, okeydoke )
            fmt.Fprintf(connection, string(original)+"\n")
          } else {
            fmt.Fprintf(connection, "ERR\r\n" )
          }
        } else {
          fmt.Println(err)
        }

      default:
	fmt.Println(emojify(requestString))
        fmt.Fprintf(connection, "%v\r\n", emojify(requestString) )
    }
    time.Sleep(600 * time.Millisecond)
  }
  connection.Close()
}

/////////////////////////////////////////////////

//  the code below is from the golang.org/pkg/net documentation.
//  while I'd written such in C after reading Richard Stevens books, I'd not done such programming in Go.
//  it's obviously very similar...with Go's go keyword analagous to Posix pthread_create.

func  server( args []string) {

  if len(args) < 3 || args[1] != "-p" {
    fmt.Fprintf(os.Stderr,"Oops: the server command arguments are bogus.\n")
    return
  }
  var fileName = args[0]
  var port     = args[2]

  //  In memory of richard stevens' books, I'm using mmap, since I know it's Unix beneath.

  fp, err := os.Open(fileName)  //  get a handle to the file to be served.
  if err != nil {
    fmt.Fprintf( os.Stderr , "Oops: could not open %v\n",fileName)
    os.Exit(1)
  }
  defer fp.Close()

  fpInfo,err := fp.Stat()  //  get the length of the file to be mmapped.
  if err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
  mmappedFileLength = fpInfo.Size()

  mmappedFile,err = syscall.Mmap(int(fp.Fd()),0,int(mmappedFileLength),syscall.PROT_READ,syscall.MAP_SHARED)
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }

  //  build a map of line number to byte offset.

  var lineNumber int = 1
  offsetOfLine[0] = 0
  for i := 0 ; i < int(mmappedFileLength) ; i++ {
    if byte(mmappedFile[i]) == '\n' {
      //    fmt.Println(i)
      offsetOfLine[lineNumber] = i + 1
      lineNumber++
    }
  }

  if verbosity {

    //  show the positions of beginnings of lines.

    var keys []int
    for k := range offsetOfLine {
      keys = append(keys,k)
    }
    sort.Ints(keys)
    for line := range keys {
      fmt.Printf("%4v ===> %v\n",line,offsetOfLine[line])
    }
  }

  ln,err := net.Listen("tcp", ":" + port )
  defer ln.Close()
  if err != nil {
    fmt.Println("the server did NOT start.")
    fmt.Println(err)
    os.Exit(1)
  }

  serverIsActive = true
  fmt.Println("about to enter the Accept-spawn loop...")
  spinnerRunning = true
  go spin_spinner()  //  give a visual indication of life.

  for serverIsActive {

    conn,err := ln.Accept()
    if err != nil {
      fmt.Println("error with connection of a client.")
    }
    go interactWithClient(conn)
  }

  spinnerRunning = false

  syscall.Munmap(mmappedFile)
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  mmappedFile = nil
}

/////////////////////////////////////////////////

func  clientHelp() {
  fmt.Printf("\n%v's read-eval-print loop\n",os.Args[0])
  fmt.Printf("  \"quit\" exits the repl,\n")
  fmt.Printf("  NNNN requests line # NNNN,\n")
  fmt.Printf("  \"shutdown\" stops the server,\n")
  fmt.Printf("  \"help\" gives these instructions,\n")
  fmt.Printf("  otherwise recognize and substitute some emoji.\n\n")
}

/////////////////////////////////////////////////

func  issueRequestFromClient( args []string ) {

  if len(args) < 1 { return }

  var hostAndPort string = args[0]

  clientHelp()
  stdin := bufio.NewReader(os.Stdin)  //  read-eval-print loop to the server.
  connection, err := net.Dial("tcp", hostAndPort )
  if err != nil {
    fmt.Printf("Oops: the client could not connect to the server.\n")
    fmt.Println(err)
    os.Exit(1)
  }
  responseReader := bufio.NewReader(connection)

  var goingStrong bool = true
  for goingStrong {
    fmt.Printf("%v > ",os.Args[0])
    command,_ := stdin.ReadString('\n')
    command    = strings.TrimRight( command , "\n" )

    switch {

      case command == "quit":
        fmt.Fprintf(connection, "QUIT\r\n")
        goingStrong = false

      case command == "shutdown":
        fmt.Fprintf(connection, "SHUTDOWN\r\n")
        goingStrong = false

      case matchesNumberRegex.MatchString( command):
        nnnn,err := strconv.Atoi(command)
        if err != nil {
          fmt.Println(err)
        } else {
          fmt.Fprintf(connection,"GET %d\r\n",nnnn)
          // The server sends a line matching "(OK|ERR)\r\n"
          response, _ := responseReader.ReadString('\n')
          response     = strings.TrimRight( response , "\r\n")
          fmt.Println(response)

          if matchesOKRegex.MatchString( response) {  //  then a second line is coming.
            response, _ = responseReader.ReadString( '\n')
            response    = strings.TrimRight( response, "\n")
          }
          fmt.Printf("\"%s\"\n",response)
        }

      default:
        fmt.Fprintf(connection, "%v\r\n" , command)
        // The server sends a line matching "(OK|ERR)\r\n"
        response, _ := responseReader.ReadString('\n')
        response     = strings.TrimRight( response , "\r\n")
        fmt.Println(response)
    }
  }
  connection.Close()
}

/////////////////////////////////////////////////

func  processCommandLine( args []string ) {

  if verbosity {

    fmt.Printf("there were %v commandline arguments.\n" , len(os.Args) )
    for i,a := range os.Args {
      fmt.Printf("   %v :=> %v\n",i,a)
    }
  }

  //  functions to call for keywords...

  keyword_handlers := map[string]func([]string) {
    "help"    : directions,
    "version" : version,
    "server"  : server,
    "serve"   : server,
    "client"  : issueRequestFromClient,
  }

  for i,arg := range args {

    if handler,ok := keyword_handlers[arg] ; ok {
      restOfArgs := args[i+1:]
      handler( restOfArgs )
      return
    } else {
      fmt.Fprintf( os.Stderr , "Oops : the monolith command \"%v\" is unknown.\n" , args[i])
      os.Exit(1)
    }
  }
}

/////////////////////////////////////////////////

func  main() {

  var args []string = os.Args[1:]

  if len(os.Args) == 1 {

    //  no commandline arguments, so show help (with no later arguments to process).

    directions( make([]string,0) )

  } else {

    //  is verbosity requested?

    for _,a := range args {
      if 0 == strings.Compare("verbose",a ) {
        fmt.Println("verbosity active")
        verbosity = true
        break
      }
    }

    processCommandLine( args )
  }
}

//func  stopwatch( f     func( []string ) ,
//                 args  []string         ,
//                 desc  string ) {
//
//  var before,after  time.Time
//
//  before = time.Now()
//  f( args )
//  after  = time.Now()
//  fmt.Printf ( "that %v took %v to run.\n" , desc , after.Sub(before) )
//}
