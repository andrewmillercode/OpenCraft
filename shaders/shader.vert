#version 330 core

layout(location = 0) in vec3 position;
layout(location = 1) in vec2 texCoord;
layout(location = 2) in float lightLevel;
out vec2 TexCoord;
out float LightLevel;

uniform mat4 projection;
uniform mat4 view;
uniform mat4 model;

void main() {
    gl_Position = projection * view * model * vec4(position, 1.0f);
    TexCoord = texCoord;
    LightLevel = lightLevel;
}