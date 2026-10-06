/**
 * Phone photos are often above the server's 5 MB photo limit. Before refusing
 * one, re-encode it on the device into the server's own box (≤ 2400 px, JPEG
 * q 0.85) — the server re-encodes every photo into that box anyway, so nothing
 * readable is lost, and EXIF (location) never leaves the phone. Returns the
 * original file when it already fits, cannot be decoded here, or the browser
 * lacks the APIs; the size check then decides.
 */
const BOX = 2400;

export async function shrinkImage(file: File, maxBytes: number): Promise<File> {
  if (file.size <= maxBytes || typeof createImageBitmap === 'undefined' || typeof document === 'undefined') return file;
  try {
    const bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' });
    const scale = Math.min(1, BOX / Math.max(bitmap.width, bitmap.height));
    const canvas = document.createElement('canvas');
    canvas.width = Math.round(bitmap.width * scale);
    canvas.height = Math.round(bitmap.height * scale);
    const ctx = canvas.getContext('2d');
    if (!ctx) return file;
    ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
    bitmap.close();
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.85));
    canvas.width = 0;
    canvas.height = 0;
    if (!blob || blob.size >= file.size) return file;
    return new File([blob], 'page.jpg', { type: 'image/jpeg' });
  } catch {
    return file;
  }
}
