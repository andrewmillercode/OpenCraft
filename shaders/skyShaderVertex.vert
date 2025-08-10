#version 410 core

// Vertex shader for rendering a skybox-style gradient.
// Expects a cube VAO with positions at layout(location=0).
// Removes camera translation so the sky stays centered on the camera.

layout(location = 0) in vec3 position;

out vec3 vDirection;

uniform mat4 projection;
uniform mat4 view;

void main() {
    // Drop translation from the view matrix so the skybox follows the camera
    mat4 viewNoTranslation = mat4(mat3(view));

    vDirection = position;

    // Position the sky at the far plane to avoid depth conflicts
    vec4 clip = projection * viewNoTranslation * vec4(position, 1.0);
    // Push to the far plane to ensure it's always behind world geometry
    clip.z = clip.w - 0.0001;

    gl_Position = clip;
}
