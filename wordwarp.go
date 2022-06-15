package main

import "regexp"
import "strings"

var sgrReset = map[*regexp.Regexp]string{
  regexp.MustCompile(`\x1b\[[3-4][0-7]m`): "\x1b[m",
  regexp.MustCompile(`\x1b\[[1-2]m`): "\x1b[22m",
  regexp.MustCompile(`\x1b\[3m`): "\x1b[23m",
  regexp.MustCompile(`\x1b\[4m`): "\x1b[24m",
}

var sgrRe *regexp.Regexp = regexp.MustCompile(
  `\x1b\[[0-9;]*m`,
)

func WordWrap(s string, limit int) (result []string) {
  words := strings.Split(s, " ")
  var wordInLine []string
  var line string
  wlc := -1

  for _, w := range words {
    wordLen := len(sgrRe.ReplaceAllString(w, ""))
    ifLen := wlc + 1 + wordLen

    if line == "" || ifLen <= limit {
      wordInLine = append(wordInLine, w)
      wlc += 1 + wordLen
    } else {
      result = append(result, line)
      wordInLine = []string{w}
      wlc = wordLen
    }
    line = strings.Join(wordInLine, " ")
  }

  result = append(result, line)
  for i, v := range result {
    n := i+1
    if n != len(result) {
      var sequel, terminator string
      for regex, rep := range sgrReset {
        isReseted := strings.Contains(v, rep)
        match := regex.FindString(v)
        if match != "" && !isReseted {
          terminator += rep
          sequel += match
        }
      }

      result[n] = sequel + result[n]
      result[i] += terminator
    }
  }
  return
}
