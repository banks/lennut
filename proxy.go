package lennut

import (
	"io"
	"log"
	"net"
	"sync/atomic"
)

func proxyBytes(stopped int32, stopCh chan struct{}, src, dst net.Conn) {
	go func() {
		defer src.Close()
		defer dst.Close()
		io.Copy(src, dst)
	}()

	go func() {
		defer src.Close()
		defer dst.Close()
		_, err := io.Copy(dst, src)
		if err != nil && atomic.LoadInt32(&stopped) == 0 {
			log.Printf("WARN dropping proxy conn: %s", err)
		}
	}()

	<-stopCh
	src.Close()
	dst.Close()
}
