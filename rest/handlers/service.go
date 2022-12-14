package handlers

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sentinel-official/cli-client/context"
	restrequests "github.com/sentinel-official/cli-client/rest/requests"
	restresponses "github.com/sentinel-official/cli-client/rest/responses"
	"github.com/sentinel-official/cli-client/services/wireguard"
	wireguardtypes "github.com/sentinel-official/cli-client/services/wireguard/types"
	clitypes "github.com/sentinel-official/cli-client/types"
	resttypes "github.com/sentinel-official/cli-client/types/rest"
	netutils "github.com/sentinel-official/cli-client/utils/net"
	restutils "github.com/sentinel-official/cli-client/utils/rest"
)

func Connect(ctx *context.ServerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			status         = clitypes.NewServiceStatus()
			statusFilePath = filepath.Join(ctx.Home(), clitypes.StatusFilename)
		)

		req, err := restrequests.NewConnect(r)
		if err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusBadRequest,
				resttypes.NewError(1001, err.Error()),
			)
			return
		}
		if err := req.Validate(); err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusBadRequest,
				resttypes.NewError(1002, err.Error()),
			)
			return
		}

		if err := status.LoadFromPath(statusFilePath); err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusInternalServerError,
				resttypes.NewError(1003, err.Error()),
			)
			return
		}

		if status.IFace != "" {
			var (
				service = wireguard.NewWireGuard().
					WithConfig(
						&wireguardtypes.Config{
							Name: status.IFace,
						},
					).
					WithHome(ctx.Home())
			)

			if service.IsUp() {
				restutils.WriteErrorToResponse(
					w, http.StatusBadRequest,
					resttypes.NewError(1004, fmt.Sprintf("service is already running on interface %s", status.IFace)),
				)
				return
			}
		}

		listenPort, err := netutils.GetFreeUDPPort()
		if err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusInternalServerError,
				resttypes.NewError(1007, err.Error()),
			)
			return
		}

		var (
			wireGuardConfig = &wireguardtypes.Config{
				Name: wireguardtypes.DefaultInterface,
				Interface: wireguardtypes.Interface{
					Addresses: []wireguardtypes.IPNet{
						{
							IP:  net.IP(req.Info[0 : 0+4]),
							Net: 32,
						},
						{
							IP:  net.IP(req.Info[4 : 4+16]),
							Net: 128,
						},
					},
					ListenPort: listenPort,
					PrivateKey: *wireguardtypes.NewKey(req.Keys[0]),
					DNS: append(
						[]net.IP{net.ParseIP("10.8.0.1")},
						req.Resolvers...,
					),
				},
				Peers: []wireguardtypes.Peer{
					{
						PublicKey: *wireguardtypes.NewKey(req.Info[26 : 26+32]),
						AllowedIPs: []wireguardtypes.IPNet{
							{IP: net.ParseIP("0.0.0.0")},
							{IP: net.ParseIP("::")},
						},
						Endpoint: wireguardtypes.Endpoint{
							Host: net.IP(req.Info[20 : 20+4]).String(),
							Port: binary.BigEndian.Uint16(req.Info[24 : 24+2]),
						},
						PersistentKeepalive: 15,
					},
				},
			}

			service = wireguard.NewWireGuard().
				WithConfig(wireGuardConfig).
				WithHome(ctx.Home())
		)

		status = clitypes.NewServiceStatus().
			WithID(req.ID).
			WithIFace(wireGuardConfig.Name)

		if err := status.SaveToPath(statusFilePath); err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusInternalServerError,
				resttypes.NewError(1008, err.Error()),
			)
			return
		}

		if err := service.PreUp(); err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusInternalServerError,
				resttypes.NewError(1009, err.Error()),
			)
			return
		}
		if err := service.Up(); err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusInternalServerError,
				resttypes.NewError(1010, err.Error()),
			)
			return
		}
		if err := service.PostUp(); err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusInternalServerError,
				resttypes.NewError(1011, err.Error()),
			)
			return
		}

		restutils.WriteResultToResponse(w, http.StatusOK, nil)
	}
}

func Disconnect(ctx *context.ServerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			status         = clitypes.NewServiceStatus()
			statusFilePath = filepath.Join(ctx.Home(), clitypes.StatusFilename)
		)

		if err := status.LoadFromPath(statusFilePath); err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusInternalServerError,
				resttypes.NewError(1001, err.Error()),
			)
			return
		}

		if status.IFace != "" {
			var (
				service = wireguard.NewWireGuard().
					WithConfig(
						&wireguardtypes.Config{
							Name: status.IFace,
						},
					).
					WithHome(ctx.Home())
			)

			if service.IsUp() {
				if err := service.PreDown(); err != nil {
					restutils.WriteErrorToResponse(
						w, http.StatusInternalServerError,
						resttypes.NewError(1002, err.Error()),
					)
					return
				}
				if err := service.Down(); err != nil {
					restutils.WriteErrorToResponse(
						w, http.StatusInternalServerError,
						resttypes.NewError(1003, err.Error()),
					)
					return
				}
				if err := service.PostDown(); err != nil {
					restutils.WriteErrorToResponse(
						w, http.StatusInternalServerError,
						resttypes.NewError(1004, err.Error()),
					)
					return
				}
			}
		}

		if err := os.Remove(statusFilePath); err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusInternalServerError,
				resttypes.NewError(1005, err.Error()),
			)
			return
		}

		restutils.WriteResultToResponse(w, http.StatusOK, nil)
	}
}

func GetStatus(ctx *context.ServerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			status         = clitypes.NewServiceStatus()
			statusFilePath = filepath.Join(ctx.Home(), clitypes.StatusFilename)
		)

		if err := status.LoadFromPath(statusFilePath); err != nil {
			restutils.WriteErrorToResponse(
				w, http.StatusInternalServerError,
				resttypes.NewError(1001, err.Error()),
			)
			return
		}

		if status.IFace != "" {
			var (
				service = wireguard.NewWireGuard().
					WithConfig(
						&wireguardtypes.Config{
							Name: status.IFace,
						},
					).
					WithHome(ctx.Home())
			)

			if service.IsUp() {
				upload, download, err := service.Transfer()
				if err != nil {
					restutils.WriteErrorToResponse(
						w, http.StatusInternalServerError,
						resttypes.NewError(1002, err.Error()),
					)
					return
				}

				restutils.WriteResultToResponse(w, http.StatusOK,
					&restresponses.GetStatus{
						ID:       status.ID,
						IFace:    status.IFace,
						Upload:   upload,
						Download: download,
					},
				)
				return
			}
		}

		restutils.WriteResultToResponse(w, http.StatusOK, nil)
	}
}
