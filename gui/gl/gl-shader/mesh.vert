#version 150

uniform mat4 Matrix;
in vec2 Position;

void main()
{
    gl_Position = Matrix * vec4(Position.xy, 0.0, 1.0);
}