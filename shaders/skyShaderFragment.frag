#version 410 core
in vec3 vDirection;
out vec4 color;

// Simple vertical gradient based on the normalized view direction's Y component.
// The top of the sky is deeper blue, fading to a lighter color near the horizon.

void main() {
    vec3 dir = normalize(vDirection);

    // Map Y from [-1, 1] to [0, 1]
    float t = clamp(dir.y * 0.5 + 0.5, 0.0, 1.0);
    t = smoothstep(0.0, 1.0, t);

    // Colors can be tuned as desired
    const vec3 topColor = vec3(0.35, 0.62, 1.00); // zenith
    const vec3 bottomColor = vec3(0.90, 0.96, 1.00); // horizon

    vec3 sky = mix(bottomColor, topColor, t);

    color = vec4(sky, 1.0);
}
