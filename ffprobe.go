package main

type Probe struct {
	Streams []struct {
		Index              int    `json:"index,omitempty"`
		CodecName          string `json:"codec_name,omitempty"`
		CodecLongName      string `json:"codec_long_name,omitempty"`
		Profile            string `json:"profile,omitempty"`
		CodecType          string `json:"codec_type,omitempty"`
		CodecTagString     string `json:"codec_tag_string,omitempty"`
		CodecTag           string `json:"codec_tag,omitempty"`
		Width              int    `json:"width,omitempty"`
		Height             int    `json:"height,omitempty"`
		CodedWidth         int    `json:"coded_width,omitempty"`
		CodedHeight        int    `json:"coded_height,omitempty"`
		ClosedCaptions     int    `json:"closed_captions,omitempty"`
		FilmGrain          int    `json:"film_grain,omitempty"`
		HasBFrames         int    `json:"has_b_frames,omitempty"`
		SampleAspectRatio  string `json:"sample_aspect_ratio,omitempty"`
		DisplayAspectRatio string `json:"display_aspect_ratio,omitempty"`
		PixFmt             string `json:"pix_fmt,omitempty"`
		Level              int    `json:"level,omitempty"`
		ChromaLocation     string `json:"chroma_location,omitempty"`
		Refs               int    `json:"refs,omitempty"`
		IsAvc              string `json:"is_avc,omitempty"`
		NalLengthSize      string `json:"nal_length_size,omitempty"`
		RFrameRate         string `json:"r_frame_rate,omitempty"`
		AvgFrameRate       string `json:"avg_frame_rate,omitempty"`
		TimeBase           string `json:"time_base,omitempty"`
		StartPts           int    `json:"start_pts,omitempty"`
		StartTime          string `json:"start_time,omitempty"`
		BitsPerRawSample   string `json:"bits_per_raw_sample,omitempty"`
		ExtradataSize      int    `json:"extradata_size,omitempty"`
		Disposition        struct {
			Default         int `json:"default,omitempty"`
			Dub             int `json:"dub,omitempty"`
			Original        int `json:"original,omitempty"`
			Comment         int `json:"comment,omitempty"`
			Lyrics          int `json:"lyrics,omitempty"`
			Karaoke         int `json:"karaoke,omitempty"`
			Forced          int `json:"forced,omitempty"`
			HearingImpaired int `json:"hearing_impaired,omitempty"`
			VisualImpaired  int `json:"visual_impaired,omitempty"`
			CleanEffects    int `json:"clean_effects,omitempty"`
			AttachedPic     int `json:"attached_pic,omitempty"`
			TimedThumbnails int `json:"timed_thumbnails,omitempty"`
			NonDiegetic     int `json:"non_diegetic,omitempty"`
			Captions        int `json:"captions,omitempty"`
			Descriptions    int `json:"descriptions,omitempty"`
			Metadata        int `json:"metadata,omitempty"`
			Dependent       int `json:"dependent,omitempty"`
			StillImage      int `json:"still_image,omitempty"`
		} `json:"disposition,omitempty"`
		Tags           *Tags  `json:"tags,omitempty"`
		SampleFmt      string `json:"sample_fmt,omitempty"`
		SampleRate     string `json:"sample_rate,omitempty"`
		Channels       int    `json:"channels,omitempty"`
		ChannelLayout  string `json:"channel_layout,omitempty"`
		BitsPerSample  int    `json:"bits_per_sample,omitempty"`
		InitialPadding int    `json:"initial_padding,omitempty"`
	} `json:"streams,omitempty"`
	Chapters []any `json:"chapters,omitempty"`
	Format   struct {
		Filename       string `json:"filename,omitempty"`
		NbStreams      int    `json:"nb_streams,omitempty"`
		NbPrograms     int    `json:"nb_programs,omitempty"`
		NbStreamGroups int    `json:"nb_stream_groups,omitempty"`
		FormatName     string `json:"format_name,omitempty"`
		FormatLongName string `json:"format_long_name,omitempty"`
		StartTime      string `json:"start_time,omitempty"`
		Duration       string `json:"duration,omitempty"`
		Size           string `json:"size,omitempty"`
		BitRate        string `json:"bit_rate,omitempty"`
		ProbeScore     int    `json:"probe_score,omitempty"`
	} `json:"format,omitempty"`
}

type Tags struct {
	VariantBitrate *string `json:"variant_bitrate,omitempty"`
	Language       *string `json:"language,omitempty"`
}
