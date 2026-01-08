package handlers

import (
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
	"github.com/photosync/server/internal/services"
)

// PublicGalleryHandler handles public gallery routes
type PublicGalleryHandler struct {
	collectionService   *services.CollectionService
	collectionRepo      repository.CollectionRepo
	collectionPhotoRepo repository.CollectionPhotoRepo
	photoRepo           repository.PhotoRepo
	storagePath         string
	templatePath        string
}

// NewPublicGalleryHandler creates a new PublicGalleryHandler
func NewPublicGalleryHandler(
	collectionService *services.CollectionService,
	collectionRepo repository.CollectionRepo,
	collectionPhotoRepo repository.CollectionPhotoRepo,
	photoRepo repository.PhotoRepo,
	storagePath string,
	templatePath string,
) *PublicGalleryHandler {
	return &PublicGalleryHandler{
		collectionService:   collectionService,
		collectionRepo:      collectionRepo,
		collectionPhotoRepo: collectionPhotoRepo,
		photoRepo:           photoRepo,
		storagePath:         storagePath,
		templatePath:        templatePath,
	}
}

// ViewGalleryBySlug serves the public gallery page by slug
func (h *PublicGalleryHandler) ViewGalleryBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "Gallery not found", http.StatusNotFound)
		return
	}

	collection, err := h.collectionService.GetCollectionBySlug(r.Context(), slug)
	if err != nil {
		if err == models.ErrCollectionNotFound || err == models.ErrCollectionAccessDenied {
			http.Error(w, "Gallery not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.renderGallery(w, r, collection)
}

// ViewGalleryByToken serves the gallery page via secret link
func (h *PublicGalleryHandler) ViewGalleryByToken(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		http.Error(w, "Gallery not found", http.StatusNotFound)
		return
	}

	collection, err := h.collectionService.GetCollectionBySecretToken(r.Context(), token)
	if err != nil {
		if err == models.ErrCollectionNotFound {
			http.Error(w, "Gallery not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.renderGallery(w, r, collection)
}

// ServeGalleryImage serves an image from a public gallery
func (h *PublicGalleryHandler) ServeGalleryImage(w http.ResponseWriter, r *http.Request) {
	photoID := chi.URLParam(r, "photoId")
	collectionID := r.URL.Query().Get("c")

	if photoID == "" || collectionID == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Verify photo is in this collection and collection is accessible
	collection, err := h.collectionRepo.GetByID(r.Context(), collectionID)
	if err != nil || collection == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Check visibility
	if collection.Visibility != models.VisibilityPublic && collection.Visibility != models.VisibilitySecretLink {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Verify photo is in collection
	inCollection, err := h.collectionPhotoRepo.IsPhotoInCollection(r.Context(), collectionID, photoID)
	if err != nil || !inCollection {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	photo, err := h.photoRepo.GetByID(r.Context(), photoID)
	if err != nil || photo == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	imagePath := filepath.Join(h.storagePath, photo.StoredPath)
	h.serveFile(w, imagePath)
}

// ServeGalleryThumbnail serves a thumbnail from a public gallery
func (h *PublicGalleryHandler) ServeGalleryThumbnail(w http.ResponseWriter, r *http.Request) {
	photoID := chi.URLParam(r, "photoId")
	collectionID := r.URL.Query().Get("c")
	size := r.URL.Query().Get("size")

	if photoID == "" || collectionID == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Verify photo is in this collection and collection is accessible
	collection, err := h.collectionRepo.GetByID(r.Context(), collectionID)
	if err != nil || collection == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Check visibility
	if collection.Visibility != models.VisibilityPublic && collection.Visibility != models.VisibilitySecretLink {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Verify photo is in collection
	inCollection, err := h.collectionPhotoRepo.IsPhotoInCollection(r.Context(), collectionID, photoID)
	if err != nil || !inCollection {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	photo, err := h.photoRepo.GetByID(r.Context(), photoID)
	if err != nil || photo == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Determine which thumbnail to serve
	var thumbPath *string
	switch size {
	case "large":
		thumbPath = photo.ThumbLarge
	case "medium":
		thumbPath = photo.ThumbMedium
	case "small", "":
		thumbPath = photo.ThumbSmall
	default:
		thumbPath = photo.ThumbSmall
	}

	// Serve thumbnail if exists
	if thumbPath != nil && *thumbPath != "" {
		fullPath := filepath.Join(h.storagePath, *thumbPath)
		if _, err := os.Stat(fullPath); err == nil {
			h.serveFile(w, fullPath)
			return
		}
	}

	// Fallback to original
	imagePath := filepath.Join(h.storagePath, photo.StoredPath)
	h.serveFile(w, imagePath)
}

// renderGallery renders the gallery HTML page
func (h *PublicGalleryHandler) renderGallery(w http.ResponseWriter, r *http.Request, collection *models.Collection) {
	// Get photos for the gallery
	photos, err := h.collectionService.GetPhotosPublic(r.Context(), collection.ID)
	if err != nil {
		http.Error(w, "Failed to load photos", http.StatusInternalServerError)
		return
	}

	// Get theme CSS
	themeCSS, err := h.collectionService.GetThemeCSS(r.Context(), string(collection.Theme))
	if err != nil {
		themeCSS = "" // Fallback to empty CSS on error
	}

	// Custom CSS
	customCSS := ""
	if collection.CustomCSS != nil {
		customCSS = *collection.CustomCSS
	}

	// Build base URL for this request
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	baseURL := scheme + "://" + r.Host

	data := models.PublicGalleryData{
		Collection: collection,
		Photos:     photos,
		ThemeCSS:   themeCSS,
		CustomCSS:  customCSS,
		BaseURL:    baseURL,
	}

	// Try to load template
	templateFile := filepath.Join(h.templatePath, "gallery", "public.html")
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		// Use embedded template
		h.renderEmbeddedGallery(w, data)
		return
	}

	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		h.renderEmbeddedGallery(w, data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to render gallery", http.StatusInternalServerError)
	}
}

// renderEmbeddedGallery renders the gallery using an embedded template
func (h *PublicGalleryHandler) renderEmbeddedGallery(w http.ResponseWriter, data models.PublicGalleryData) {
	tmpl := template.Must(template.New("gallery").Parse(embeddedGalleryTemplate))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to render gallery", http.StatusInternalServerError)
	}
}

// serveFile serves a file with proper content type
func (h *PublicGalleryHandler) serveFile(w http.ResponseWriter, path string) {
	file, err := os.Open(path)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Detect content type
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	contentType := http.DetectContentType(buffer[:n])

	file.Seek(0, 0)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	io.Copy(w, file)
}

// Embedded gallery template for when no file template exists
const embeddedGalleryTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Collection.Name}}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }

        {{.ThemeCSS}}

        body {
            font-family: var(--font-family, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif);
            background-color: var(--bg-color, #0f172a);
            color: var(--text-color, #f1f5f9);
            min-height: 100vh;
        }

        .header {
            padding: 40px 20px;
            text-align: center;
            border-bottom: 1px solid var(--border-color, #334155);
        }

        .header h1 {
            font-size: 2.5rem;
            margin-bottom: 10px;
            color: var(--text-color);
        }

        .header p {
            color: var(--text-muted, #94a3b8);
            max-width: 600px;
            margin: 0 auto;
        }

        .gallery {
            padding: 20px;
            max-width: 1600px;
            margin: 0 auto;
        }

        .photo-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
            gap: 16px;
        }

        .photo-card {
            position: relative;
            aspect-ratio: 1;
            overflow: hidden;
            border-radius: 8px;
            background-color: var(--card-color, #1e293b);
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }

        .photo-card:hover {
            transform: translateY(-4px);
            box-shadow: 0 10px 40px rgba(0,0,0,0.3);
        }

        .photo-card img {
            width: 100%;
            height: 100%;
            object-fit: cover;
        }

        /* Lightbox */
        .lightbox {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0,0,0,0.95);
            z-index: 1000;
            justify-content: center;
            align-items: center;
        }

        .lightbox.active {
            display: flex;
        }

        .lightbox img {
            max-width: 90vw;
            max-height: 90vh;
            object-fit: contain;
        }

        .lightbox-close {
            position: absolute;
            top: 20px;
            right: 20px;
            font-size: 40px;
            color: white;
            cursor: pointer;
            z-index: 1001;
        }

        .lightbox-nav {
            position: absolute;
            top: 50%;
            transform: translateY(-50%);
            font-size: 60px;
            color: white;
            cursor: pointer;
            padding: 20px;
            user-select: none;
        }

        .lightbox-prev { left: 10px; }
        .lightbox-next { right: 10px; }

        .photo-count {
            text-align: center;
            padding: 20px;
            color: var(--text-muted);
        }

        {{.CustomCSS}}
    </style>
</head>
<body>
    <header class="header">
        <h1>{{.Collection.Name}}</h1>
        {{if .Collection.Description}}<p>{{.Collection.Description}}</p>{{end}}
    </header>

    <main class="gallery">
        <div class="photo-grid">
            {{range $i, $photo := .Photos}}
            <div class="photo-card" data-index="{{$i}}" onclick="openLightbox({{$i}})">
                <img src="/gallery/photos/{{$photo.ID}}/thumbnail?c={{$.Collection.ID}}&size=medium"
                     alt="Photo" loading="lazy">
            </div>
            {{end}}
        </div>

        <p class="photo-count">{{len .Photos}} photos</p>
    </main>

    <div class="lightbox" id="lightbox">
        <span class="lightbox-close" onclick="closeLightbox()">&times;</span>
        <span class="lightbox-nav lightbox-prev" onclick="prevPhoto()">&#10094;</span>
        <img id="lightbox-img" src="" alt="Full size">
        <span class="lightbox-nav lightbox-next" onclick="nextPhoto()">&#10095;</span>
    </div>

    <script>
        const photos = [
            {{range $i, $photo := .Photos}}
            {id: "{{$photo.ID}}", collectionId: "{{$.Collection.ID}}"}{{if lt $i (len $.Photos)}},{{end}}
            {{end}}
        ];

        let currentIndex = 0;

        function openLightbox(index) {
            currentIndex = index;
            updateLightboxImage();
            document.getElementById('lightbox').classList.add('active');
            document.body.style.overflow = 'hidden';
        }

        function closeLightbox() {
            document.getElementById('lightbox').classList.remove('active');
            document.body.style.overflow = '';
        }

        function updateLightboxImage() {
            const photo = photos[currentIndex];
            document.getElementById('lightbox-img').src =
                '/gallery/photos/' + photo.id + '/image?c=' + photo.collectionId;
        }

        function nextPhoto() {
            currentIndex = (currentIndex + 1) % photos.length;
            updateLightboxImage();
        }

        function prevPhoto() {
            currentIndex = (currentIndex - 1 + photos.length) % photos.length;
            updateLightboxImage();
        }

        document.addEventListener('keydown', (e) => {
            if (!document.getElementById('lightbox').classList.contains('active')) return;

            if (e.key === 'Escape') closeLightbox();
            if (e.key === 'ArrowRight') nextPhoto();
            if (e.key === 'ArrowLeft') prevPhoto();
        });

        document.getElementById('lightbox').addEventListener('click', (e) => {
            if (e.target.id === 'lightbox') closeLightbox();
        });
    </script>
</body>
</html>`
