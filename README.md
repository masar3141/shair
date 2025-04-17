<table>
  <tr>
    <td><img src="./img/sender.gif" alt="Sender demo" /></td>
    <td><img src="./img/receiver.gif" alt="Receiver demo" /></td>
  </tr>
</table>

**Shair** is a terminal-based interface for sending files between two machines with zero configuration.

It uses mDNS to automatically discover peers on the local network and transfers files over a custom TCP protocol—no pairing, or setup required.  
Just grab the binary for your platform from the releases section, make it available in your `PATH`, and start it on both machines.

Once running, each instance automatically discovers others on the network.  
The sender selects a peer and enters file paths to send.  
The receiver is prompted with a transfer request and can choose to accept or reject it.  
On acceptance, the files are streamed directly to the destination.

## Roadmap

### Core
- [x] Zero-conf file transfer over local network
- [ ] Add tests
- [ ] Add direct send with password (SCP-style; sender enters receiver’s predefined password, no manual acceptance required)
- [ ] Support Bluetooth discovery and Airdrop compatibility (owl)
- [ ] Remote discovery support (outside local network)

### Security
-  [ ] TLS encryption for local TCP file transfers

### UI
- [ ] Graphical UI for desktop
- [ ] Mobile UI for phone-to-phone or phone-to-desktop transfers


## Inspiration
- [magic-wormhole](https://github.com/magic-wormhole/magic-wormhole)
- [LocalSend](https://github.com/localsend/localsend) 

## Disclaimer

This project is an early-stage prototype built as a demonstration for a university application. 
It is nowhere to be close to production ready.

- Files are transferred in plain text (no encryption)
- The code may contain bugs 
- Maintenance and updates are not guaranteed
