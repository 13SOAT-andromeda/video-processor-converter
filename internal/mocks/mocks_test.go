package mocks_test

import (
	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/mocks"
)

// Garante em tempo de compilação que cada mock satisfaz seu port.
var (
	_ ports.ObjectStorage   = (*mocks.MockObjectStorage)(nil)
	_ ports.VideoProber     = (*mocks.MockVideoProber)(nil)
	_ ports.FrameExtractor  = (*mocks.MockFrameExtractor)(nil)
	_ ports.Archiver        = (*mocks.MockArchiver)(nil)
	_ ports.StatusPublisher = (*mocks.MockStatusPublisher)(nil)
)
