package main


var re * regexp.Regexp = regexp.MustCompile("[Hh]amburger")

func emojify( line string ) string {
  return re.ReplaceAllString(line,"🍔")
}

func main() {

  scanner := bufio.NewScanner(os.Stdin)

  for scanner.Scan() {
    line := scanner.Text()
    fmt.Println( emojify(line) )
  }
}
