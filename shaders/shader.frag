#version 330 core

in float LightLevel;
in vec2 TexCoord;
out vec4 color;

uniform sampler2D textureSampler;

void main() {
    color = texture(textureSampler, TexCoord)* vec4(0.06 * LightLevel,0.06 * LightLevel,0.06 * LightLevel,1.0);
    //color = vec4(0.0,1.0,0.0,0.5) * vec4(0.05,0.05,0.05,1.0);
}