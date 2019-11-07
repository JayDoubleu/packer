package hcl2template

// PackerConfig represents a loaded packer config
type PackerConfig struct {
	Sources map[SourceRef]*Source

	Variables PackerV1Variables

	Builds Builds

	Communicators map[CommunicatorRef]*Communicator
}
