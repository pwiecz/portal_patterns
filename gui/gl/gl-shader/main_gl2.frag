#version 120

uniform sampler2D Texture;

varying vec2 Frag_UV;
varying vec4 Frag_Color;

void main()
{
    gl_FragColor = vec4(Frag_Color.rgb, Frag_Color.a * texture2D(Texture, Frag_UV.st).r);
}