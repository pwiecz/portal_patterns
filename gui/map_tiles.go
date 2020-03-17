package main

import "errors"
import "fmt"
import "image"
import "image/png"
import "io/ioutil"
import "log"
import "net/http"
import "os"
import "path"
import "strconv"
import "sync"
//import "time"
import "github.com/golang/groupcache/lru"
import "github.com/pwiecz/atk/tk"

const (
	MAX_DOWNLOAD_THREADS = 2
)
type tileCoord struct {
	x, y, zoom int
}

type SyncCache struct {
	cache *lru.Cache
	lock  sync.Mutex
}

func NewSyncCache(numEntries int) *SyncCache {
	return &SyncCache{cache: lru.New(numEntries)}
}
func (c *SyncCache) Add(key lru.Key, value interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache.Add(key, value)
}
func (c *SyncCache) Get(key lru.Key) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.cache.Get(key)
}
func (c *SyncCache) Remove(key lru.Key) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache.Remove(key)
}

type MapTiles struct {
	cacheDir         string
	memCache         *SyncCache
	fetchSemaphore   chan empty
	requestsInFlight map[tileCoord]bool
	onTileRead       func(tileCoord, *tk.Image)
}

func NewMapTiles() *MapTiles {
	cacheDirBase, err := os.UserCacheDir()
	cacheDir := ""
	if err == nil {
		cacheDir = path.Join(cacheDirBase, "portal_patterns")
	}
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		os.MkdirAll(cacheDir, 0755)
	}
	memCache := NewSyncCache(1000)
	semaphore := make(chan empty, 50)
	e := empty{}
	for i := 0; i < MAX_DOWNLOAD_THREADS; i++ {
		semaphore <- e
	}
	requestsInFlight := make(map[tileCoord]bool)
	return &MapTiles{
		cacheDir:         cacheDir,
		memCache:         memCache,
		fetchSemaphore:   semaphore,
		requestsInFlight: requestsInFlight}
}

func (m *MapTiles) GetTile(coord tileCoord) (*tk.Image, bool) {
	if coord.zoom < 0 || coord.y < 0 {
		log.Println("Negative tile coordinates", coord)
		return nil, false
	}
	if coord.zoom > 20 {
		log.Println("Too high zoom factor", coord)
		return nil, false
	}
	maxCoord := 1 << coord.zoom
	if coord.y >= maxCoord {
		log.Println("Too high x,y coords", coord)
		return nil, false
	}
	wrappedCoord := coord
	for wrappedCoord.x < 0 {
		wrappedCoord.x += maxCoord
	}
	wrappedCoord.x %= maxCoord
	if tile, ok := m.memCache.Get(wrappedCoord); ok {
		if tileImage, ok := tile.(*tk.Image); ok {
			return tileImage, true
		}
		m.memCache.Remove(wrappedCoord)
	}
	if _, ok := m.requestsInFlight[wrappedCoord]; !ok {
		m.requestsInFlight[wrappedCoord] = true
		go func(coord tileCoord) {
			img, err := m.getTileSlow(wrappedCoord)
			tk.Async(func() {
				delete(m.requestsInFlight, wrappedCoord)
				if err != nil {
					m.onTileRead(coord, nil)
					return
				}

				tkImg := tk.NewImage()
				tkImg.SetImage(img)
				m.memCache.Add(wrappedCoord, tkImg)
				m.onTileRead(coord, tkImg)
			})
		}(coord)
	}
	if wrappedCoord.zoom > 0 {
		zoomedOutCoord := tileCoord{x: wrappedCoord.x / 2, y: wrappedCoord.y / 2, zoom: wrappedCoord.zoom - 1}
		if tile, ok := m.memCache.Get(zoomedOutCoord); ok {
			if tileImage, ok := tile.(*tk.Image); ok {
				sourceX := (wrappedCoord.x % 2) * 128
				sourceY := (wrappedCoord.y % 2) * 128
				tkImg := tk.NewImage()
				tkImg.Copy(tileImage, tk.ImageCopyAttrFrom(sourceX, sourceY, sourceX+128, sourceY+128), tk.ImageCopyAttrZoom(2.0, 2.0))
				return tkImg, false
			}
		}
	}
	return nil, false
}

func (m *MapTiles) SetOnTileRead(onTileRead func(tileCoord, *tk.Image)) {
	m.onTileRead = onTileRead
}
func (m *MapTiles) getTileSlow(coord tileCoord) (image.Image, error) {
	if m.cacheDir == "" {
		return m.fetchTile(coord)
	}
	cachedTileDir := path.Join(m.cacheDir, strconv.Itoa(coord.zoom), strconv.Itoa(coord.x))
	if _, err := os.Stat(cachedTileDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cachedTileDir, 0755); err != nil {
			log.Println("Cannot create cache dir", err)
			return m.fetchTile(coord)
		}
	}
	cachedTilePath := path.Join(cachedTileDir, strconv.Itoa(coord.y)+".png")
	f, err := os.Open(cachedTilePath)
	if err == nil {
		img, err := png.Decode(f)
		f.Close()
		if err == nil {
			return img, err
		}
		log.Println("Cannot decode cached file", cachedTilePath, err)
		if err := os.Remove(cachedTilePath); err != nil {
			log.Println("Cannot remove cached tile", cachedTilePath)
		}
	}
	img, err := m.fetchTile(coord)
	if err != nil {
		return nil, err
	}
	tmpfile, err := ioutil.TempFile(cachedTileDir, ".tile_*.png")
	if err != nil {
		log.Println("Cannot create temp tile file", err)
		return img, nil
	}
	tmpname := tmpfile.Name()
	if err := png.Encode(tmpfile, img); err != nil {
		log.Println("Cannot encode image tile file", err)
		tmpfile.Close()
		os.Remove(tmpname)
		return img, nil
	}
	if err := tmpfile.Sync(); err != nil {
		log.Println("Cannot sync temp file", err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Println("Cannot close temp file", err)
	}
	if err := os.Rename(tmpname, cachedTilePath); err != nil {
		log.Println("Cannot rename temp file", err)
	}
	return img, nil
}

func (m *MapTiles) fetchTile(coord tileCoord) (image.Image, error) {
	select {
	case <-m.fetchSemaphore:
		defer func() { 
			m.fetchSemaphore <- empty{} }()
		url := fmt.Sprintf("http://a.tile.openstreetmap.org/%d/%d/%d.png", coord.zoom, coord.x, coord.y)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "portal_patterns 4.0")
		var client http.Client
//		client.Timeout = 10 * time.Second
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("\"%s\" returned status code %d", url, resp.StatusCode)
		}
		tile, err := png.Decode(resp.Body)
		if err != nil {
			return nil, err
		}
		return tile, nil
	default:
		return nil, errors.New("Too many simultaneous get requests")
	}
}
