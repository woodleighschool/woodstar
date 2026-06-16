package storage

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

func ServeContent(ctx huma.Context, presigner Presigner, key string, opts GetOptions) {
	url, err := presigner.PresignGet(ctx.Context(), key, 0, opts)
	if err != nil {
		ctx.SetStatus(http.StatusInternalServerError)
		return
	}
	ctx.SetHeader("Location", url)
	ctx.SetStatus(http.StatusFound)
}
