# Transcode REST

Very simple API to transcode files or ffmpeg supported muxers (like HLS or MPEG-DASH)

## Config

Everything is optional

- STORAGE_DIRECTORY will store transcoded files with hash of the URL provided, otherwise a tmp dir will be used
- FFPROBE_PATH to overwrite `$PATH` binary of `ffmpeg`
- FFMPEG_PATH to overwrite `$PATH` binary of `ffprobe`
- LISTEN_PORT of the API otherwise `:8080`

## API

- `/` for health checks
- `/transcode?url=http%3A%2F%2Fgoodone.fr%2Fhls.m3u8` to transcode the HLS to a sing mp4 file
- `/pdf/info?url=http%3A%2F%2Fgoodone.fr%2Fhls.m3u8` to get pdf informations
- `/pdf/thumbnail?page=0&url=http%3A%2F%2Fgoodone.fr%2Fhls.m3u8` to convert a pdf page into a .jpg image
- `/video/thumbnail?url=http%3A%2F%2Fgoodone.fr%2Fhls.m3u8` to get the first frame in .jpg from video
- `/blur?url=http%3A%2F%2Fgoodone.fr%2Fhls.m3u8` to generate a blur hash from an image
