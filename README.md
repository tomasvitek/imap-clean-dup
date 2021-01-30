# IMAP clean duplicate emails

This is a simple go script used to remove duplicates from specifies IMAP mailbox. It is based on [project by Elias Norberg](https://github.com/yzzyx/imap-clean-dup).

## Usage

Run `go run main.go -server imap.gmail.com -username username@gmail.com -password "mypassword123" -mbox "Agenda" -ignore-message-id -dry-run -list-only-dups`.

### Params

- `-username`: IMAP user (required)
- `-password`: IMAP password (required)
- `-server`: IMAP server (required)
- `-mbox`: Mailbox to remove duplicates from (required)
- `-list-only-dups`: If present, only duplicated messages are output
- `-ignore-message-id`: If present, MessageId is ignored, a hash for each message is instead calculated
- `-dry-run`: If present, no removal will be performed

## Gotchas

When running, make sure that the imap server is set to move messages to bin or delete when message is marked as deleted over imap. Otherwise, it will only be moved to archive, not deleted. 

In Gmail's settings this is in `Forwarding and POP/IMAP` under `When a message is marked as deleted and expunged from the last visible IMAP folder` section.

