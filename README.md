# ngress


 Support:
   - websockets, http2, http3
   - static files hosting
   - upstreams to -> services, stattic-files and UNIX sockets
   - DaemonSet with hostNetwork: true
   - tested with https://cert-manager.io


# simple deploy
kubectl apply -f ./deployments/ngress.yaml

