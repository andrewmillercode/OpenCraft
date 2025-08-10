#version 410 core

in float LightLevel;
in vec2 TexCoord;
in vec3 TextureTint;
in vec2 OverlayCoord;
out vec4 color;

uniform sampler2D texture0;
float light;
float minBrightness = 1.0;

void main() {
    vec4 baseTexture = texture(texture0, TexCoord);
    light = (LightLevel + minBrightness) / 15.0;

    if (OverlayCoord.x != 0) {
        color = baseTexture;
        vec4 overlayTexture = texture(texture0, OverlayCoord);
        overlayTexture.rgb *= TextureTint;
        color = mix(color, overlayTexture, overlayTexture.a);
    } else {
        color = baseTexture * vec4(TextureTint[0], TextureTint[1], TextureTint[2], 1.0);
    }

    color *= vec4(light, light, light, 1.0);
}
