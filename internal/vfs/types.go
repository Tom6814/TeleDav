package vfs

import "telegram-webdav/internal/store"

type DirectoryListing struct {
	Directories []store.Directory `json:"directories"`
	Files       []store.FileEntry `json:"files"`
}
