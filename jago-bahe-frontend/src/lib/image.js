/**
 * Downscale and re-encode a photo so it fits the API's request-body cap.
 *
 * Why this exists: a photo travels inline as a base64 data URL, and base64
 * inflates it by ~4/3. The backend caps a whole request body at 1 MiB
 * (pkg/httpx/response.go maxBodyBytes), so a phone-camera photo — routinely
 * 2–5MB, i.e. 2.7–6.8MB encoded — was a guaranteed 413. Reporting a problem with
 * a photo simply failed, and the form reported it as a generic "could not
 * submit". Compressing here is what the Design Guideline (A.6, "lazy-load/
 * compress evidence photos") asked for from the start.
 *
 * The cap is not the thing to raise: it is a deliberate memory-exhaustion guard
 * (B8), and a 5MB photo per report is not a size this platform needs. A civic
 * evidence photo has to show a pothole, not survive a crop to print.
 */

/**
 * Data-URL budget for one photo, comfortably inside the server's 1 MiB body cap
 * once the title, description, location and ids are accounted for.
 */
export const MAX_IMAGE_BYTES = 900 * 1024

/**
 * Long edge, in pixels. Ample for evidence — it shows a pothole, it is not a
 * print master — and the UI renders it far smaller still.
 *
 * Measured, not guessed: at 1600px a pathological 12MP photo (pure noise, which
 * JPEG cannot compress) landed at 888KB against the 900KB budget even at the
 * lowest quality step — passing, but with 1% to spare, so a merely very detailed
 * photo could tip over and be refused. 1280px is 64% of the pixels and restores
 * real headroom without costing any detail that matters here.
 */
const MAX_DIMENSION = 1280

/** Tried in order until the encoded photo fits MAX_IMAGE_BYTES. */
const QUALITY_STEPS = [0.82, 0.7, 0.6, 0.5, 0.4]

function loadImage(file) {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const img = new Image()
    img.onload = () => {
      URL.revokeObjectURL(url)
      resolve(img)
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('could not decode image'))
    }
    img.src = url
  })
}

/**
 * Compress `file` to a JPEG data URL within MAX_IMAGE_BYTES.
 *
 * Re-encodes to JPEG regardless of input type: a PNG photo from a screenshot or
 * a modern HEIC-converted upload can be many times larger than the same image as
 * JPEG, and transparency is meaningless for a photograph.
 *
 * @param {File} file
 * @returns {Promise<string>} a `data:image/jpeg;base64,...` URL
 * @throws if the image cannot be decoded, or cannot be squeezed under the budget
 */
export async function compressImage(file) {
  const img = await loadImage(file)

  // Only ever scale down — enlarging a small photo would add bytes and no detail.
  const scale = Math.min(1, MAX_DIMENSION / Math.max(img.width, img.height))
  const canvas = document.createElement('canvas')
  canvas.width = Math.max(1, Math.round(img.width * scale))
  canvas.height = Math.max(1, Math.round(img.height * scale))

  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('canvas unavailable')
  ctx.drawImage(img, 0, 0, canvas.width, canvas.height)

  for (const quality of QUALITY_STEPS) {
    const dataUrl = canvas.toDataURL('image/jpeg', quality)
    if (dataUrl.length <= MAX_IMAGE_BYTES) return dataUrl
  }

  // Every step overshot: refuse rather than post a body the server will reject.
  // The caller turns this into a message the reporter can act on.
  throw new Error('image too large after compression')
}
