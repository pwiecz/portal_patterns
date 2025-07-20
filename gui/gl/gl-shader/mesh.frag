#version 150

uniform vec4 Color;
uniform float AAWidth;

in float Frag_Dist;

out vec4 Out_Color;

void main()
{
   float opacity = Color.a;
   if (AAWidth > 0 && abs(Frag_Dist) > 1.0 - AAWidth) {
      opacity = smoothstep(Color.a, 0, clamp(AAWidth - 1.0 + abs(Frag_Dist), 0.0, 1.0) / AAWidth);
   }
   Out_Color = vec4(Color.xyz, opacity);
}