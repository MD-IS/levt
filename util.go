package main

import "io"
import "os"
import "path"
import "bufio"
import "os/exec"
import "io/ioutil"

type Openable interface {
  Open() (io.ReadCloser, error)
}

func open(href string) error {
  itemFile, ok := epubContent[href]
  if ok && itemFile != nil {
    tmpSRC := path.Join(os.TempDir(), path.Base(href))
    reader, err := itemFile.Open()
    if err != nil {return err}
    SaveTMP(reader, tmpSRC)
    reader.Close()

    return exec.Command("xdg-open", tmpSRC).Run()
  }
  return exec.Command("xdg-open", href).Run()
}

func openReader(href string) (io.ReadCloser, error) {
  if i, ok := epubContent[href]; ok {
    return i.Open()
  }
  return os.Open(href)
}

func SaveTMP(r io.Reader, s string) error {
  file, err := os.Create(s)
  if err != nil {return err}
  _, err = bufio.NewReader(r).WriteTo(file)
  file.Close()
  return err
}

func ReadContent(f Openable) []byte {
  reader, err := f.Open()
  Catch(err)

  contents, err := ioutil.ReadAll(reader)
  Catch(err)

  reader.Close()
  return contents
}

func Catch(err error) {
  if err != nil {panic(err)}
}
