#version 150

uniform sampler2D Texture;
uniform vec4 Color;
uniform float PxRange;

in vec2 Frag_UV;

out vec4 Out_Color;

float median(float r, float g, float b) {
    return max(min(r, g), min(max(r, g), b));
}

void main() {
    vec3 msd = texture2D(Texture, Frag_UV).rgb;
    float sd = median(msd.r, msd.g, msd.b);
    float screenPxDistance = PxRange*(sd - 0.5);
    float opacity = clamp(screenPxDistance + 0.5, 0.0, 1.0);
    Out_Color = vec4(Color.rgb, opacity);
}
