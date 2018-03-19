package main

import (
	"io"
	"fmt"
	"bufio"
	"strings"
	"net"
	"sync"
	"sync/atomic"
	"sort"
	//"regexp"
	//"encoding/json"
)

// глобальные переменные запрещены
// cgo запрещен

func CheckBrowser(s string) bool {
	cnt := strings.Count(s, "60.0.3112.90")
	
	if cnt >= 3 {
		return true
	}
	
	cnt += strings.Count(s, "52.0.2743.116")
	
	if cnt >= 3 {
		return true
	}
	
	cnt += strings.Count(s, "57.0.2987.133")
	
	if cnt >= 3 {
		return true
	}
	
	return false
}

type user struct {
	Name string `json:"name"`
	Email string `json:"email"`
}

func Fast(in io.Reader, out io.Writer, networks []string) {
	i := 0
	bf := bufio.NewScanner(in)
	var cnt int64
	
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	
	nets := make([]*net.IPNet, 0)
	fnets := make([]string, 0)
	
	for _, ip := range networks {
		if strings.HasSuffix(ip, "0/16") {
			cutIp := strings.TrimSuffix(ip, ".0/16")
			dotI := strings.LastIndex(cutIp, ".")
			fnets = append(fnets, cutIp[0:dotI])
		} else if strings.HasSuffix(ip, "0/24") {
			fnets = append(fnets, strings.TrimSuffix(ip, ".0/24"))
		} else {
			_, nip, _ := net.ParseCIDR(ip)
			nets = append(nets, nip)
		}
	}
	
	ww := make(map[int]string)
	
	for bf.Scan() {
		i ++
		sft := bf.Text()
	
		wg.Add(1)

		go func (s string, i int) {
			defer wg.Done()
			
			if !CheckBrowser(s) {
				return
			}
			
			//bytIp := r.FindAllString(s, -1)
			i1 := strings.Index(s, "\"hits\":[")
			i2 := strings.Index(s, "],\"job\":")
			
			ipsList := s[i1+8:i2]
			bytIp := strings.Split(strings.Replace(ipsList, "\"", "", -1), ",")
			
			cntIp := 0
			ok := false
			
		IPCHECK:
			for _, ip := range bytIp {
				for _, sip := range fnets {
					if strings.HasPrefix(ip, sip) {
						cntIp ++
						
						if cntIp >= 3 {
							ok = true
							atomic.AddInt64(&cnt, 1)
							break IPCHECK
						}
					}
				}
				
				ipcidr := net.ParseIP(ip)
				
				for _, ipnetcird := range nets {
					if ipnetcird.Contains(ipcidr) {
						cntIp ++
						
						if cntIp >= 3 {
							ok = true
							atomic.AddInt64(&cnt, 1)
							break IPCHECK
						}
					}
				}
			}
			
			if !ok {
				return
			}
			
			//u := user{}
			//json.Unmarshal([]byte(s), &u)
			j2 := i1-2
			j1 := strings.Index(s[0:j2], "\"email\":")
			
			email := s[j1+9:j2]
			
			j1 = strings.Index(s[i2:], "\"name\":")
			j2 = strings.Index(s[i2:], ",\"phone\":")
			
			name  := s[i2+j1+8:i2+j2-1]
			
			mu.Lock()
			ww[i] = fmt.Sprintf("\n[%d] %s <%s>", i, name, strings.Replace(email, "@", " [at] ", 1))
			mu.Unlock()
		}(sft, i)
	}
	
	wg.Wait()
	
	fmt.Fprintf(out, "Total: %d", cnt)
	
	keys := make([]int, len(ww))
	j := 0
	for k := range ww {
		keys[j] = k
		j ++
	}
	sort.Ints(keys)
	
	for _, k := range keys {
		fmt.Fprint(out, ww[k])
	}
	
	fmt.Fprint(out, "\n")
}
