CLImage is a CLI tool, written in modern Go that generates images from text prompts.

Add the following header to every source file with comments, followed by a blank line:
```
CLImage is a AI image generation CLI tool.
Copyright (C) 2025  Mayer & Ott GbR

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public Licen
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```

It is similar to [SST OpenCode](https://github.com/sst/opencode) but for images.

Make sure every cobra command has a short and long description.

The AI providers should be abstracted to enable adding new providers easily.
Every model has its own settings.
