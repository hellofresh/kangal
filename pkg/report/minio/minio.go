package report

import (
	"archive/tar"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/minio/minio-go/v6"
)

const (
	pathSeparator = "/"
)

//MinioFileSystem struct to work with MinioFileSystem lib backend as a http.Filesystem
type MinioFileSystem struct {
	*minio.Client
	Bucket string
}

// Open - implements http.Filesystem implementation.
func (m *MinioFileSystem) Open(name string) (http.File, error) {
	if strings.HasSuffix(name, pathSeparator) {
		return &minioFile{
			client: m.Client,
			object: nil,
			isDir:  true,
			bucket: m.Bucket,
			prefix: strings.TrimSuffix(name, pathSeparator),
		}, nil
	}

	name = strings.TrimPrefix(name, pathSeparator)
	parts := strings.Split(name, "/")
	if 1 == len(parts) {
		parts = append(parts, "index.html")
	}
	name = strings.Join(parts, "/")

	loadTestFile := parts[0]

	loadTestObj, err := getObject(m, loadTestFile)
	if err == os.ErrNotExist {
		return m.fallbackOpen(name)
	}
	if err != nil {
		return nil, os.ErrNotExist
	}

	loadTestInfo, _ := loadTestObj.Stat()

	if "application/x-tar" == loadTestInfo.ContentType {
		tarName := strings.TrimPrefix(name, loadTestFile+"/")
		tarReader := tar.NewReader(loadTestObj)

		for {
			tarHeader, err := tarReader.Next()
			if nil != err {
				break
			}

			if tarName != tarHeader.Name {
				continue
			}

			return &memoryFile{
				at:      0,
				name:    tarName,
				read:    tarReader,
				size:    tarHeader.Size,
				modTime: tarHeader.ModTime,
			}, nil
		}

		return nil, os.ErrNotExist
	}

	return &minioFile{
		client: m.Client,
		object: loadTestObj,
		isDir:  false,
		bucket: m.Bucket,
		prefix: loadTestFile,
	}, nil
}

func (m *MinioFileSystem) fallbackOpen(name string) (http.File, error) {
	obj, err := getObject(m, name)
	if err != nil {
		return nil, os.ErrNotExist
	}

	return &minioFile{
		client: m.Client,
		object: obj,
		isDir:  false,
		bucket: m.Bucket,
		prefix: name,
	}, nil
}

func getObject(m *MinioFileSystem, name string) (*minio.Object, error) {
	obj, err := m.Client.GetObject(m.Bucket, name, minio.GetObjectOptions{})
	if err == nil {
		if _, err = obj.Stat(); err == nil {
			return obj, nil
		}
	}
	return nil, os.ErrNotExist
}

// objectInfo implements os.FileInfo interface,
// is returned during Readdir(), Stat() operations.
type objectInfo struct {
	minio.ObjectInfo
	prefix string
	isDir  bool
}

//Name ...
func (o objectInfo) Name() string {
	return o.ObjectInfo.Key
}

//Size ...
func (o objectInfo) Size() int64 {
	return o.ObjectInfo.Size
}

//Mode ...
func (o objectInfo) Mode() os.FileMode {
	if o.isDir {
		return os.ModeDir
	}
	return os.FileMode(0644)
}

//ModTime ...
func (o objectInfo) ModTime() time.Time {
	return o.ObjectInfo.LastModified
}

//IsDir ...
func (o objectInfo) IsDir() bool {
	return o.isDir
}

//Sys ...
func (o objectInfo) Sys() interface{} {
	return &syscall.Stat_t{}
}

// A minioFile implements http.File interface, returned by a MinioFileSystem
// Open method and can be served by the FileServer implementation.
type minioFile struct {
	client *minio.Client
	object *minio.Object
	bucket string
	prefix string
	isDir  bool
}

//Close ...
func (h *minioFile) Close() error {
	return h.object.Close()
}

//Read ...
func (h *minioFile) Read(p []byte) (n int, err error) {
	return h.object.Read(p)
}

//Seek ...
func (h *minioFile) Seek(offset int64, whence int) (int64, error) {
	return h.object.Seek(offset, whence)
}

//Readdir ...
func (h *minioFile) Readdir(count int) ([]os.FileInfo, error) {
	// List 'N' number of objects from a Bucket-name with a matching prefix.
	listObjectsN := func(bucket, prefix string, count int) (objsInfo []minio.ObjectInfo, err error) {
		// Create a done channel to control 'ListObjects' go routine.
		doneCh := make(chan struct{})

		// Free the channel upon return.
		defer close(doneCh)

		i := 1
		for object := range h.client.ListObjects(bucket, prefix, false, doneCh) {
			if object.Err != nil {
				return nil, object.Err
			}
			i++
			// Verify if we have printed N objects.
			if i == count {
				return
			}
			objsInfo = append(objsInfo, object)
		}
		return objsInfo, nil
	}

	// List non-recursively first count entries for prefix 'prefix" prefix.
	objsInfo, err := listObjectsN(h.bucket, h.prefix, count)
	if err != nil {
		return nil, os.ErrNotExist
	}
	var fileInfos []os.FileInfo
	for _, objInfo := range objsInfo {
		if strings.HasSuffix(objInfo.Key, pathSeparator) {
			fileInfos = append(fileInfos, objectInfo{
				ObjectInfo: minio.ObjectInfo{
					Key:          strings.TrimSuffix(objInfo.Key, pathSeparator),
					LastModified: objInfo.LastModified,
				},
				prefix: strings.TrimSuffix(objInfo.Key, pathSeparator),
				isDir:  true,
			})
			continue
		}
		fileInfos = append(fileInfos, objectInfo{
			ObjectInfo: objInfo,
		})
	}
	return fileInfos, nil
}

//Stat ...
func (h *minioFile) Stat() (os.FileInfo, error) {
	if h.isDir {
		return objectInfo{
			ObjectInfo: minio.ObjectInfo{
				Key:          h.prefix,
				LastModified: time.Now().UTC(),
			},
			prefix: h.prefix,
			isDir:  true,
		}, nil
	}

	objInfo, err := h.object.Stat()
	if err != nil {
		return nil, os.ErrNotExist
	}

	return objectInfo{
		ObjectInfo: objInfo,
	}, nil
}
