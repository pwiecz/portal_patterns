package osm

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

const (
	MAX_DOWNLOAD_THREADS = 2
	MAX_ZOOM_LEVEL       = 19
)

var ErrBusy = errors.New("Too many simultaneous requests")

type TileCoord struct {
	X, Y, Zoom int
}

type empty struct{}
type MapTiles struct {
	cacheDir              string
	fetchSemaphore        chan empty
	requestsInFlightMutex sync.Mutex
	requestsInFlight      map[TileCoord]empty
}

func NewMapTiles() *MapTiles {
	cacheDirBase, err := os.UserCacheDir()
	cacheDir := ""
	if err == nil {
		cacheDir = filepath.Join(cacheDirBase, "portal_patterns")
	}
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		os.MkdirAll(cacheDir, 0755)
	}
	semaphore := make(chan empty, 50)
	e := empty{}
	for i := 0; i < MAX_DOWNLOAD_THREADS; i++ {
		semaphore <- e
	}
	requestsInFlight := make(map[TileCoord]empty)
	return &MapTiles{
		cacheDir:         cacheDir,
		fetchSemaphore:   semaphore,
		requestsInFlight: requestsInFlight}
}

func (m *MapTiles) GetTile(coord TileCoord) (image.Image, error) {
	if coord.Zoom < 0 || coord.Y < 0 {
		return nil, fmt.Errorf("Negative tile coordinates %v", coord)
	}
	if coord.Zoom > MAX_ZOOM_LEVEL {
		return nil, fmt.Errorf("Too high zoom factor %v", coord)
	}
	maxCoord := 1 << coord.Zoom
	if coord.Y >= maxCoord {
		return nil, fmt.Errorf("Invalid x,y coords %v", coord)
	}
	wrappedCoord := coord
	for wrappedCoord.X < 0 {
		wrappedCoord.X += maxCoord
	}
	wrappedCoord.X %= maxCoord

	m.requestsInFlightMutex.Lock()
	if _, ok := m.requestsInFlight[wrappedCoord]; ok {
		m.requestsInFlightMutex.Unlock()
		return nil, fmt.Errorf("Coords already requested %v", wrappedCoord)
	}
	m.requestsInFlight[wrappedCoord] = empty{}
	m.requestsInFlightMutex.Unlock()

	img, err := m.getTileSlow(wrappedCoord)

	m.requestsInFlightMutex.Lock()
	delete(m.requestsInFlight, wrappedCoord)
	m.requestsInFlightMutex.Unlock()

	if err != nil {
		return nil, err
	}
	return img, nil
}

func (m *MapTiles) getTileSlow(coord TileCoord) (image.Image, error) {
	if m.cacheDir == "" {
		return m.fetchTile(coord)
	}
	cachedTileDir := filepath.Join(m.cacheDir, strconv.Itoa(coord.Zoom), strconv.Itoa(coord.X))
	if _, err := os.Stat(cachedTileDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cachedTileDir, 0755); err != nil {
			log.Println("Cannot create cache dir", err)
			return m.fetchTile(coord)
		}
	}
	cachedTilePath := filepath.Join(cachedTileDir, strconv.Itoa(coord.Y)+".png")
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

func (m *MapTiles) fetchTile(coord TileCoord) (image.Image, error) {
	select {
	case <-m.fetchSemaphore:
		defer func() {
			m.fetchSemaphore <- empty{}
		}()
		url := fmt.Sprintf("http://a.tile.openstreetmap.org/%d/%d/%d.png", coord.Zoom, coord.X, coord.Y)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "portal_patterns 4.0")
		var client http.Client
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
		return nil, ErrBusy
	}
}
