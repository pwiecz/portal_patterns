#version 150

uniform mat4 Matrix;

in vec2 Position;
in vec2 UV;

out vec2 Frag_UV;

void main()
{
	gl_Position = Matrix * vec4(Position.xy, 0.0, 1.0);
	Frag_UV = UV;
}