#version 150

uniform vec4 Color;
uniform float AAWidth;

in float Frag_Dist;

out vec4 Out_Color;

void main()
{
   float opacity = smoothstep(1.0, 1.0 - AAWidth, abs(Frag_Dist));
   Out_Color = vec4(Color.xyz, mix(0.0, Color.a, opacity));
}