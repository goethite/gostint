/*
Copyright 2018 Graham Lee Bevan <graham.bevan@ntlworld.com>

This file is part of gostint.

gostint is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

gostint is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with gostint.  If not, see <https://www.gnu.org/licenses/>.
*/

package cleanup

import (
	"context"
	"log"
	"math/rand"
	"time"

	client "docker.io/go-docker"
	"docker.io/go-docker/api/types"
)

// docker image ID -> datetime last used in a job
var imageMap = make(map[string]time.Time)

// ImageUsed to capture when an image was last used in a job
func ImageUsed(id string, when time.Time) {
	imageMap[id] = when
}

// Images cleans up unused docker images
func Images() {
	ctx := context.Background()
	for {
		cli, err := client.NewEnvClient()
		if err != nil {
			log.Printf("get docker client error: %s", err)
		}

		imgList, err := cli.ImageList(ctx, types.ImageListOptions{
			All: true,
		})
		if err != nil {
			log.Printf("cleanup images get images from docker error: %s", err)
		}
		for _, img := range imgList {
			age := time.Since(imageMap[img.ID])
			if age.Hours() > 24 {
				log.Printf("Removing unused image %s: %s", img.ID, img.RepoTags)
				imagesDeleted, err := cli.ImageRemove(ctx, img.ID, types.ImageRemoveOptions{})
				if err != nil {
					log.Printf("Remove docker image error: %s", err)
				}
				log.Printf("Images Removed: %v", imagesDeleted)
			}
		}

		// wait 1 min + random upto another 1 min (splay)
		r := rand.Intn(int(time.Minute))
		time.Sleep(time.Duration(r)*time.Nanosecond + time.Minute)
	} // for
}

func imageList(ctx context.Context, cli *client.Client) {
	// Get list of images on host
	imgList, err := cli.ImageList(ctx, types.ImageListOptions{
		All: true,
	})
	if err != nil {
		log.Printf("cleanup images get images from docker error: %s", err)
	}
	for _, img := range imgList {
		log.Printf(
			"img.RepoTags: %s Created: %v, Last Used: %s",
			img.RepoTags,
			time.Unix(img.Created, 0),
			imageMap[img.ID],
		)
	}
}
