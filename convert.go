package main

import "encoding/xml"

func convertSUB(d *xml.Decoder) (res string) {
  outer: for t, _ := d.Token(); t != nil; {
    switch token := t.(type) {
      case xml.CharData: {
        for _, c := range token {
          switch c {
            case '1': {res += "₁"}
            case '2': {res += "₂"}
            case '3': {res += "₃"}
            case '4': {res += "₄"}
            case '5': {res += "₅"}
            case '6': {res += "₆"}
            case '7': {res += "₇"}
            case '8': {res += "₈"}
            case '9': {res += "₉"}
            case '0': {res += "₀"}
          }
        }
      }

      case xml.StartElement: {d.Skip()}
      case xml.EndElement: {break outer}
    }

    t, _ = d.Token()
  }
  return
}

func convertSUP(d *xml.Decoder) (res string) {
  outer: for t, _ := d.Token(); t != nil; {
    switch token := t.(type) {
      case xml.CharData: {
        for _, c := range token {
          switch c {
            case '1': {res += "¹"}
            case '2': {res += "²"}
            case '3': {res += "³"}
            case '4': {res += "⁴"}
            case '5': {res += "⁵"}
            case '6': {res += "⁶"}
            case '7': {res += "⁷"}
            case '8': {res += "⁸"}
            case '9': {res += "⁹"}
            case '0': {res += "⁰"}
          }
        }
      }

      case xml.StartElement: {d.Skip()}
      case xml.EndElement: {break outer}
    }

    t, _ = d.Token()
  }
  return
}
