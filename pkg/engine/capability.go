package engine

// HasLogicalBackup checks whether an adaptor supports logical backup.
func HasLogicalBackup(a Adaptor) (LogicalBackupAdaptor, bool) {
	lb, ok := a.(LogicalBackupAdaptor)
	return lb, ok
}

// HasPhysicalBackup checks whether an adaptor supports physical backup.
func HasPhysicalBackup(a Adaptor) (PhysicalBackupAdaptor, bool) {
	pb, ok := a.(PhysicalBackupAdaptor)
	return pb, ok
}

// HasFileBackup checks whether an adaptor supports file-level backup.
func HasFileBackup(a Adaptor) (FileBackupAdaptor, bool) {
	fb, ok := a.(FileBackupAdaptor)
	return fb, ok
}

// HasTransfer checks whether an adaptor supports table transfer.
func HasTransfer(a Adaptor) (TransferAdaptor, bool) {
	ta, ok := a.(TransferAdaptor)
	return ta, ok
}

// HasQuery checks whether an adaptor supports raw SQL execution.
func HasQuery(a Adaptor) (QueryAdaptor, bool) {
	qa, ok := a.(QueryAdaptor)
	return qa, ok
}
