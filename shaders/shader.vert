#version 330 core

layout(location = 0) in vec3 position;
layout(location = 1) in vec2 texCoord;
layout(location = 2) in float lightLevel;
layout(location = 3) in vec3 textureTint;
layout(location = 4) in vec2 overlayCoord;
out vec2 TexCoord;
out vec2 OverlayCoord;
out float LightLevel;
out vec3 TextureTint;

uniform mat4 projection;
uniform mat4 view;
uniform mat4 model;

void main() {
    gl_Position = projection * view * model * vec4(position, 1.0f);
    TexCoord = texCoord;
    LightLevel = lightLevel;
    TextureTint = textureTint;
    OverlayCoord = overlayCoord;
}