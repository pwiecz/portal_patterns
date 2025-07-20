#version 150

uniform mat4 Matrix;

in vec2 Position;
in float Dist;

out float Frag_Dist;

void main()
{
    gl_Position = Matrix * vec4(Position.xy, 0.0, 1.0);
    Frag_Dist = Dist;
}